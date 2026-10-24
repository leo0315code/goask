# GoAsk

基于 **Go + MySQL** 的问答社区系统（类 Stack Overflow），自带完整 Web 页面，单个二进制开箱部署。

## 功能

- **Web 页面**：首页（最新/热门/未回答/标签/搜索 + 分页）、问题详情（投票/评论/回答/采纳/编辑删除）、提问、登录注册、个人主页（声望与提问列表）—— Vue 3 构建产物嵌入 Go 二进制，单文件部署
- 用户：注册 / 登录（JWT）/ 资料管理 / 声望与计数
- 问题：发布、编辑、软删除、关闭、最新 / 热门 / 未回答 / 按标签列表
- 回答：发布、编辑、删除、提问者采纳（可更换）
- 投票：问题与回答 ±1，可切换撤销，计数与声望事务内一致更新
- 评论：问题 / 回答下的评论
- 标签：自动创建、热门排序
- 检索：标题 + 正文中文全文检索（MySQL ngram）
- 筛选：排序（最新/热门/未回答）+ 时间范围（一天/一周/一月/一年）+ 标签 + 全文搜索
- 防滥用：注册图形验证码、注册开关与邀请码、同 IP 每日注册上限、发帖频次限制（最小间隔 + 24h 上限）、重复内容拦截、双层 IP 限流
- 生产要素：健康检查、优雅停机、按 IP 限流、统一错误码、结构化日志、自动数据库迁移

## 技术栈

Go 1.22+ · Gin · GORM · MySQL 8 · JWT · bcrypt · slog · Vue 3 · Vite · Docker Compose

## 快速开始

### 方式一：Docker Compose（推荐）

```bash
cp .env.example .env        # 修改 JWT_SECRET 等配置
docker compose -f deployments/docker-compose.yml up -d --build
# 浏览器打开 http://localhost:8080
```

镜像构建分两阶段：Node 编译前端 → Go 编译并嵌入前端产物。应用启动时自动执行 `migrations/` 下的数据库迁移。

### 方式二：本地运行

前置：MySQL 8 + Node 18+（构建页面）+ Go 1.22+：

```bash
mysql -h 127.0.0.1 -uroot -p -e "CREATE DATABASE IF NOT EXISTS goask DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci"

cp .env.example .env        # 填写 DB_USER / DB_PASSWORD / JWT_SECRET
make build                  # 构建前端 + 后端 → bin/goask
./bin/goask                 # 启动，浏览器打开 http://localhost:8080
```

### 前端独立开发（热更新）

```bash
cd web && npm install
npm run dev                 # http://localhost:5173，/api 自动代理到 :8080 的后端
```

## 演示账号（本地演示数据）

`leo` / `alice` / `bob`，密码均为 `secret123`。

更多 API 用例见 [docs/api.md](docs/api.md) 第 9 节的 curl 冒烟脚本。

## 配置项（环境变量）

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `APP_ENV` | `dev` | `release` 时 Gin 进入生产模式 |
| `PORT` | `8080` | HTTP 监听端口 |
| `DB_HOST` / `DB_PORT` | `127.0.0.1` / `3306` | MySQL 地址 |
| `DB_USER` / `DB_PASSWORD` | `root` / 空 | MySQL 凭据 |
| `DB_NAME` | `goask` | 库名 |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` | `25` / `10` | 连接池 |
| `DB_CONN_MAX_LIFETIME` | `5m` | 连接复用时长 |
| `JWT_SECRET` | **必填** | 签名密钥（≥16 字符，缺失时拒绝启动） |
| `JWT_EXPIRE` | `72h` | 令牌有效期 |
| `BCRYPT_COST` | `12` | 密码哈希强度 |
| `CORS_ORIGINS` | `*` | 逗号分隔白名单，生产必须收敛 |
| `RATE_LIMIT_GLOBAL_RPM` | `120` | 全局限流（次/分/IP） |
| `RATE_LIMIT_AUTH_RPM` | `10` | 注册/登录限流（次/分/IP） |
| `REGISTER_ENABLED` | `true` | 是否开放注册（关闭后注册页显示提示） |
| `REGISTER_INVITE_CODE` | 空 | 设置后注册必须携带邀请码 |
| `REGISTER_MAX_PER_IP_PER_DAY` | `3` | 同 IP 每日注册上限（0 = 不限） |
| `MIGRATE_ON_START` | `true` | 启动时自动执行迁移 |
| `SEED_ADMIN_USERNAME` / `SEED_ADMIN_PASSWORD` | 空 | 设置后启动时确保管理员存在 |

## 项目结构

```
goask/
├── cmd/server/          # 入口：装配依赖、优雅停机
├── internal/
│   ├── config/          # 环境变量配置
│   ├── handler/         # HTTP 协议层（参数绑定、响应封装）
│   ├── service/         # 业务规则与编排（依赖 repository 接口，可单测）
│   ├── repository/      # GORM 数据访问 + 事务一致性 + 迁移器
│   ├── model/           # 数据模型与状态常量
│   ├── middleware/      # 认证 / CORS / 限流 / 日志 / 恢复
│   └── pkg/             # apperr 错误码、httpx 响应、jwtx、passwordx
├── web/                 # Vue 3 + Vite 前端（src/ 源码，dist/ 构建产物 embed 进二进制）
├── migrations/          # SQL 迁移文件（启动时自动按序执行）
├── docs/                # 架构 / 数据库 / API 设计文档
├── deployments/         # docker-compose.yml
├── Dockerfile
└── Makefile
```

## 常用命令

```bash
make run      # 本地启动
make build    # 编译到 bin/
make test     # 单元测试
make vet      # 静态检查
make up       # docker compose 启动
make down     # docker compose 停止
```

## 文档

- [架构设计](docs/design.md)：分层、选型、关键决策（投票一致性、热度公式、限流）
- [数据库设计](docs/database.md)：ER、表结构、索引与全文检索、迁移策略
- [API 设计](docs/api.md)：全部端点、请求/响应示例、错误码

## 上线检查清单

- [ ] `APP_ENV=release`，`JWT_SECRET` 为强随机值（≥32 字符）
- [ ] `CORS_ORIGINS` 收敛为正式域名
- [ ] MySQL 单独部署并定期备份；`schema_migrations` 保证迁移幂等
- [ ] 反向代理（Nginx/SLB）终止 TLS，`TRUSTED_PROXIES` 配置为代理网段
- [ ] LB 健康检查指向 `/readyz`，容器健康检查指向 `/healthz`
