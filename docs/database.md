# GoAsk 数据库设计文档

> MySQL 8.0+，库名 `goask`，字符集 `utf8mb4` / `utf8mb4_unicode_ci`，所有表 `InnoDB`。

## 1. ER 关系

```
users 1──* questions 1──* answers
  │            │ 1            │ 1
  │            │ └──* comments│ └──* comments
  │            │ 1            │ 1
  │            └──* votes     └──* votes        （votes: 多态指向 question/answer）
  │            *──1 tags via question_tags (m:n)
  └──* comments / votes
```

- `votes`、`comments` 采用多态设计（`target_type` + `target_id`），避免 question/answer 各建一套同构表；
- `question_tags` 为 `questions` 与 `tags` 的多对多关联表。

## 2. 表结构

### 2.1 users（用户）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | BIGINT UNSIGNED PK | 雪花前时代直接自增即可，预留 BIGINT 空间 |
| username | VARCHAR(32) UNIQUE | 3–32 位字母/数字/下划线，注册时校验 |
| email | VARCHAR(128) UNIQUE | 登录凭据之一 |
| password_hash | VARCHAR(255) | bcrypt 哈希，永不外泄 |
| avatar_url / bio | VARCHAR | 资料项 |
| role | VARCHAR(16) | `user` / `admin` |
| status | VARCHAR(16) | `active` / `banned` |
| reputation | INT | 声望：回答赞 +10、问题赞 +5、踩 -2、采纳 +15，撤销对称回退 |
| question_count / answer_count | INT | 冗余计数，个人主页免 COUNT(*) |

索引：`uk_users_username`、`uk_users_email`。

### 2.2 questions（问题）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | BIGINT UNSIGNED PK | |
| user_id | BIGINT UNSIGNED | 提问者 |
| title | VARCHAR(255) | 必填，10–150 字符 |
| content | MEDIUMTEXT | 正文（Markdown 纯文本存储），上限 64KB |
| status | VARCHAR(16) | `open` / `closed` / `deleted`（软删除） |
| view_count | INT UNSIGNED | 浏览数，详情页异步 +1 |
| answer_count | INT UNSIGNED | 冗余计数（不含已删除回答） |
| upvote_count / downvote_count | INT | 冗余计数 |
| accepted_answer_id | BIGINT UNSIGNED NULL | 已采纳回答；`NULL` 未采纳 |
| last_activity_at | DATETIME(3) | 出现新回答/新回答被采纳时刷新，默认排序键 |

索引：
- `idx_questions_user(user_id)`：我的提问；
- `idx_questions_status_activity(status, last_activity_at)`：列表默认排序覆盖索引；
- `ft_questions_text(title, content) WITH PARSER ngram`：中文全文索引。

> 说明：全文索引与 `status` 过滤组合时，MySQL 会选择 FULLTEXT 访问路径再过滤 status，检索结果集可控时性能足够。

### 2.3 answers（回答）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | BIGINT UNSIGNED PK | |
| question_id | BIGINT UNSIGNED | 外键 → questions.id |
| user_id | BIGINT UNSIGNED | 回答者 |
| content | MEDIUMTEXT | |
| status | VARCHAR(16) | `published` / `deleted` |
| upvote_count / downvote_count | INT | 冗余计数 |
| is_accepted | TINYINT(1) | 采纳标记，与 questions.accepted_answer_id 双写、事务内保持一致 |

索引：`idx_answers_question(question_id, status)`、`idx_answers_user(user_id)`。

### 2.4 votes（投票，多态）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | BIGINT UNSIGNED PK | |
| user_id | BIGINT UNSIGNED | 投票人 |
| target_type | VARCHAR(16) | `question` / `answer` |
| target_id | BIGINT UNSIGNED | 问题或回答 ID |
| value | TINYINT | `1` 赞 / `-1` 踩 |

索引：`uk_votes_user_target(user_id, target_type, target_id)` —— 一人对一目标最多一票，同时支撑 upsert 语义；`idx_votes_target(target_type, target_id)` 供计数回溯。

### 2.5 comments（评论，多态）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | BIGINT UNSIGNED PK | |
| user_id | BIGINT UNSIGNED | |
| target_type | VARCHAR(16) | `question` / `answer` |
| target_id | BIGINT UNSIGNED | |
| content | VARCHAR(1000) | |
| status | VARCHAR(16) | `published` / `deleted` |

索引：`idx_comments_target(target_type, target_id, id)` —— 按目标翻页免 filesort。

### 2.6 tags / question_tags（标签）

- `tags(id, name UNIQUE, description, question_count)`：`question_count` 冗余计数支撑热门标签排序；
- `question_tags(question_id, tag_id)` 复合主键 + `idx_question_tags_tag(tag_id)` 反向索引，支撑「按标签取问题」「标签问题计数」。

### 2.7 schema_migrations（迁移版本）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| version | VARCHAR(64) PK | 迁移文件名（如 `0001_init`） |
| applied_at | DATETIME(3) | 应用时间 |

## 3. 设计要点

### 3.1 冗余计数器的一致性
`answer_count`、`upvote_count`、`downvote_count`、`question_count`、`tag.question_count` 均为冗余字段，用于列表/个人页免聚合查询。规则：

- **所有写入点收敛在 repository 层的事务方法内**（投票、发布/删除回答、采纳），业务层无法绕过；
- 计数使用 `UPDATE ... SET cnt = cnt + 1` 原子自增，不读改写；
- 已提供 `tools/recount.go` 思路（文档级）：极端漂移时可用 SQL 全量重算（见 3.4）。

### 3.2 软删除
内容删除统一置 `status='deleted'`，保留数据可审计、可恢复；唯一例外是 votes 随目标删除仅做计数对齐。列表查询全部带 status 过滤。

### 3.3 为什么不用外键级联
仅保留少量强一致外键（answers→questions、question_tags→两表），删除走软删除而非物理删除，故不使用 `ON DELETE CASCADE`；votes/comments 多态目标无法建外键，一致性由 service 层存在性校验兜底。

### 3.4 容量与扩展预估
- 千万级 questions：`idx_questions_status_activity` 覆盖首页/最新列表；翻页深度 >100 页时建议切换游标分页（`WHERE id < ? ORDER BY id DESC LIMIT n`，repo 层已留参数化基础）；
- 全文检索 QPS 升高或需要纠错/同义词时引入 ES，同步链路走「写库后异步投递」；
- 声望/计数如需极致写入吞吐，可引入 Redis 缓冲 + 定时回刷（Roadmap）。

## 4. 迁移策略

- 迁移文件：`migrations/NNNN_描述.up.sql` / `.down.sql`，按文件名序号执行；
- 应用启动时自动执行（`MIGRATE_ON_START=true`），记录到 `schema_migrations`，已应用的跳过 —— **幂等、可重放**；
- 变更纪律：只增不改；需要回滚时手写对应 `.down.sql` 并评审（v1 未提供自动 down 执行入口，避免误操作）。
