-- 回滚楼中楼字段
ALTER TABLE comments
  DROP KEY idx_comments_parent,
  DROP COLUMN parent_id,
  DROP COLUMN reply_to_user_id;
