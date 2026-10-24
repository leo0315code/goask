-- 站内通知表
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS notifications (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL COMMENT '接收人',
  type VARCHAR(32) NOT NULL COMMENT 'answer/comment_question/comment_answer/accept/vote_up_question/vote_up_answer',
  actor_id BIGINT UNSIGNED NOT NULL COMMENT '触发人',
  target_type VARCHAR(16) NOT NULL DEFAULT '',
  target_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  question_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用于跳转的问题 ID',
  content VARCHAR(500) NOT NULL DEFAULT '' COMMENT '摘要（问题标题/内容片段）',
  is_read TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_notifications_user (user_id, is_read, id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
