-- 评论支持楼中楼：parent_id 指向根评论（0 = 根），reply_to_user_id 记录被回复人
SET NAMES utf8mb4;

ALTER TABLE comments
  ADD COLUMN parent_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER target_id,
  ADD COLUMN reply_to_user_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER parent_id,
  ADD KEY idx_comments_parent (parent_id);
