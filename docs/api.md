# GoAsk API 设计文档

> Base URL：`http://localhost:8080/api/v1`
> Content-Type：`application/json`；时间统一 ISO8601（UTC）。

## 1. 认证

登录/注册成功后返回 JWT，后续请求携带请求头：

```
Authorization: Bearer <token>
```

- 必须登录的接口：发布/编辑/删除内容、投票、采纳、评论、修改资料；
- 可选登录的接口：问题详情（登录后返回当前用户投票状态）。

## 2. 统一响应

```json
// 成功
{"code": 0, "message": "ok", "data": {}}
// 失败
{"code": 40000, "message": "参数错误: title 为必填", "data": null}
```

错误码：`0` 成功；`40000` 参数错误；`40100` 未登录/token 失效；`40300` 无权限；`40400` 不存在；`40900` 状态冲突；`42200` 业务校验失败；`42900` 限流；`50000` 内部错误。

分页请求：`?page=1&page_size=20`（`page_size` 上限 100）；分页响应：

```json
{"code":0,"message":"ok","data":{"list":[],"total":123,"page":1,"page_size":20}}
```

限流：全局 120 req/min/IP，注册/登录 10 req/min/IP，超限返回 HTTP 429。

### 防灌水（发布频次）

发布问题 / 回答 / 评论时服务端强制检查，违规返回 `42900` 或 `42200`：

| 类型 | 最小间隔 | 24 小时上限 | 重复内容 |
| --- | --- | --- | --- |
| 问题 | 60 秒 | 20 | 同一用户 24h 内相同标题拒绝 |
| 回答 | 30 秒 | 50 | 24h 内相同正文拒绝 |
| 评论 | 10 秒 | 200 | 24h 内相同正文拒绝 |

## 3. 健康检查

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/healthz` | 存活探针，不查库 |
| GET | `/readyz` | 就绪探针，`SELECT 1` |

## 4. 认证模块

### GET `/api/v1/auth/captcha`

注册前获取图形算术验证码（一次性，5 分钟有效）：

```json
{"code":0,"message":"ok","data":{
  "captcha_id":"a1b2c3...",
  "svg":"<svg ...>4 + 3 = ?</svg>",
  "register_enabled": true,
  "invite_required": false
}}
```

### POST `/api/v1/auth/register`

```json
// 请求（captcha_id / captcha_code 必填，来自 captcha 接口）
{"username": "leo", "email": "leo@example.com", "password": "secret123",
 "captcha_id": "a1b2c3...", "captcha_code": "7", "invite_code": ""}
// 200
{"code":0,"message":"ok","data":{"token":"eyJ...","user":{"id":1,"username":"leo","email":"leo@example.com"}}}
```

规则：username 3–32 位 `[A-Za-z0-9_]`；密码 8–72 位；重复 → `40900`。
注册防护（触发时返回对应错误码）：

| 防护 | 条件 | 错误 |
| --- | --- | --- |
| 注册开关 | `REGISTER_ENABLED=false` | `40300` |
| 邀请码 | 配置了 `REGISTER_INVITE_CODE` 且不匹配 | `40300` |
| 验证码 | 缺失 / 错误 / 过期（一次性） | `42200` |
| IP 限额 | 同 IP 当日注册 ≥ `REGISTER_MAX_PER_IP_PER_DAY`（默认 3） | `42900` |
| 认证限流 | 注册/登录 > 10 次/分/IP | `42900` |

### POST `/api/v1/auth/login`

```json
// 请求（email 或 username 二选一传）
{"account": "leo", "password": "secret123"}
// 200 同 register
```

失败返回 `42200`「账号或密码错误」，不区分两种情况。被封禁用户返回 `40300`。

### GET `/api/v1/users/me`　（需登录）

返回当前用户完整资料（含声望、提问/回答计数）。

### PUT `/api/v1/users/me`　（需登录）

```json
{"bio": "Go 开发者", "avatar_url": "https://..."}
```

### GET `/api/v1/users/:username`

公开主页：资料 + 计数（不含 email）。

### PUT `/api/v1/users/me/password`　（需登录）

```json
{"old_password": "secret123", "new_password": "newpass456"}
```

校验原密码；新密码 8–72 字节。

## 4.5 通知模块（需登录）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/notifications?unread=1&page=` | 通知列表（带 `unread_count`），触发人昵称与问题标题联出 |
| POST | `/api/v1/notifications/:id/read` | 单条已读 |
| POST | `/api/v1/notifications/read-all` | 全部已读 |

触发时机：有人回答/评论你的内容、你的回答被采纳、有人赞同你的问题或回答（撤销与反对不通知，自己操作自己不通知）。

### GET `/api/v1/notifications/stream?token=<JWT>`　（SSE 实时推送）

`text/event-stream` 长连接；通知产生时服务端即时推送：

```
data: {"type":"answer","content":"gin 框架怎么做优雅停机？","question_id":6}
```

EventSource 无法携带请求头，JWT 通过查询参数传递；25s 心跳保活，客户端断线自动重连。

## 4.6 管理端（仅管理员）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/admin/stats` | 概览统计（用户/问题/回答/评论/投票总数 + 今日新增） |
| GET | `/api/v1/admin/users?q=&page=` | 用户列表（用户名/邮箱模糊搜索） |
| PUT | `/api/v1/admin/users/:id/status` | `{"status":"banned"|"active"}` 封禁/解封 |
| GET | `/api/v1/admin/questions?q=&status=all\|open\|closed\|deleted&page=` | 问题管理列表（含已删除） |
| GET | `/api/v1/admin/comments?q=&page=` | 评论管理列表（含已删除） |

### DELETE `/api/v1/comments/:id`　（需登录）

删除评论（软删除）：作者本人或管理员。

规则：不能封禁自己和管理员账号；**封禁后该用户已签发的 token 立即失效**（必登录接口每次从数据库校验状态与最新角色）。

## 5. 问题模块

### GET `/api/v1/questions`

Query：`page`、`page_size`、`sort`（`latest` 默认 / `hot` / `unanswered`）、`tag`（标签名）、`time`（时间过滤：`day` / `week` / `month` / `year`，缺省全部）。

```json
{"code":0,"message":"ok","data":{
  "list":[{"id":1,"title":"如何用 Go 连接 MySQL？","tags":[{"id":1,"name":"go"}],
           "author":{"id":1,"username":"leo"},"answer_count":2,"view_count":10,
           "upvote_count":3,"status":"open","created_at":"2026-10-24T08:00:00Z"}],
  "total":123,"page":1,"page_size":20}}
```

### POST `/api/v1/questions`　（需登录）

```json
{"title":"如何用 Go 连接 MySQL？","content":"如题，想用 database/sql……","tags":["go","mysql"]}
```

校验：标题 10–150 字符；正文非空 ≤65536 字节；标签 0–5 个（每个 ≤32 字符，不存在则创建）。

### GET `/api/v1/questions/:id`

详情；登录时额外返回 `viewer_vote`（`0/1/-1`）。浏览数 +1。

### PUT `/api/v1/questions/:id`　（仅作者）

同创建字段；问题已有回答仍可编辑（编辑不重置活动时间）。

### DELETE `/api/v1/questions/:id`　（作者或管理员）

软删除，返回 `204` 语义（`code=0`）。

### POST `/api/v1/questions/:id/accept`　（仅提问者）

```json
{"answer_id": 42}
```

采纳/更换采纳；回答必须属于该问题且未删除。事务内切换 `accepted_answer_id`、`is_accepted` 并调整声望。

### 评论

- `GET /api/v1/questions/:id/comments`　分页
- `POST /api/v1/questions/:id/comments`　（需登录）`{"content": "...", "parent_id": 0}` ≤1000 字符

楼中楼：请求可带 `parent_id`（根评论 ID）发表回复，服务端拍平到根（只允许一层嵌套）；回复通知被回复的评论作者。列表接口返回树形结构：根评论分页（旧→新），每条根评论带 `replies[]` 与 `reply_to_name`。

## 6. 回答模块

### GET `/api/v1/questions/:id/answers`

Query：`page`、`page_size`、`sort`（`latest` 默认 / `votes`）；被采纳回答恒排最前（`sort=latest` 时也置顶）。

### POST `/api/v1/questions/:id/answers`　（需登录）

```json
{"content": "用 gorm.io/driver/mysql ……"}
```

事务内：创建回答 + `answer_count`+1 + 刷新问题活动时间。以下情况返回 `42200`：提问者回答自己的问题、问题已关闭、**问题已采纳答案（已解决）**。

**问题状态语义**：`开放中`（无采纳）→ `已解决`（已采纳，不再接受新回答，评论可继续）→ `已关闭`（回答与评论全部锁定）。采纳与关闭均有前端二次确认。

### PUT `/api/v1/answers/:id`　（仅作者）/ DELETE `/api/v1/answers/:id`　（作者或管理员，软删除并回减计数）

### 评论

- `GET /api/v1/answers/:id/comments`
- `POST /api/v1/answers/:id/comments`

## 7. 投票

### POST `/api/v1/votes`　（需登录）

```json
// 请求：target_type ∈ {"question","answer"}；value ∈ {1,-1}
{"target_type": "answer", "target_id": 42, "value": 1}
// 200：返回切换后的状态；value=0 表示该票已被撤销
{"code":0,"message":"ok","data":{"target_type":"answer","target_id":42,
  "value":1,"upvote_count":3,"downvote_count":0}}
```

语义：未投过 → 投；同向再点 → 撤销（返回 value=0）；反向 → 切换。

## 8. 标签与检索

### GET `/api/v1/tags`

按 `question_count` 降序：`{"code":0,"message":"ok","data":[{"id":1,"name":"go","question_count":12}]}`

### GET `/api/v1/search?q=关键词&page=1`

标题 + 正文中文全文检索（ngram，最小 2 字符），按相关度排序，返回结构同问题列表。

## 9. curl 冒烟示例

```bash
BASE=http://localhost:8080/api/v1

# 注册并拿 token
TOKEN=$(curl -s $BASE/auth/register -d '{"username":"leo","email":"leo@ex.com","password":"secret123"}' \
  -H 'Content-Type: application/json' | jq -r .data.token)

# 提问
QID=$(curl -s $BASE/questions -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"title":"Go 如何连接 MySQL 数据库？","content":"想用 database/sql 或 gorm，求推荐","tags":["go","mysql"]}' | jq -r .data.id)

# 回答并采纳
AID=$(curl -s $BASE/questions/$QID/answers -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"content":"推荐 gorm，gorm.io/gorm 安装后用 driver/mysql 打开连接"}' | jq -r .data.id)
curl -s $BASE/questions/$QID/accept -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"answer_id\":$AID}"

# 检索
curl -s "$BASE/search?q=mysql"
```
