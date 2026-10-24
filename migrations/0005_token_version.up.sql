-- Token 版本号：改密码 / 全端登出时 +1，旧 JWT 立即失效
SET NAMES utf8mb4;

ALTER TABLE users
  ADD COLUMN token_version BIGINT NOT NULL DEFAULT 0;
