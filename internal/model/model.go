// Package model 定义数据库模型与状态常量，字段与 migrations/ 中的 DDL 一一对应。
package model

import "time"

const (
	RoleUser  = "user"
	RoleAdmin = "admin"

	UserStatusActive = "active"
	UserStatusBanned = "banned"

	QuestionStatusOpen    = "open"
	QuestionStatusClosed  = "closed"
	QuestionStatusDeleted = "deleted"

	AnswerStatusPublished = "published"
	AnswerStatusDeleted   = "deleted"

	CommentStatusPublished = "published"
	CommentStatusDeleted   = "deleted"

	TargetQuestion = "question"
	TargetAnswer   = "answer"
)

type User struct {
	ID            uint64    `gorm:"primaryKey;column:id" json:"id"`
	Username      string    `gorm:"column:username" json:"username"`
	Email         string    `gorm:"column:email" json:"email"`
	PasswordHash  string    `gorm:"column:password_hash" json:"-"`
	AvatarURL     string    `gorm:"column:avatar_url" json:"avatar_url"`
	Bio           string    `gorm:"column:bio" json:"bio"`
	Role          string    `gorm:"column:role" json:"role"`
	Status        string    `gorm:"column:status" json:"status"`
	Reputation    int       `gorm:"column:reputation" json:"reputation"`
	QuestionCount int       `gorm:"column:question_count" json:"question_count"`
	AnswerCount   int       `gorm:"column:answer_count" json:"answer_count"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (User) TableName() string { return "users" }

type Tag struct {
	ID            uint64    `gorm:"primaryKey;column:id" json:"id"`
	Name          string    `gorm:"column:name" json:"name"`
	Description   string    `gorm:"column:description" json:"description"`
	QuestionCount int       `gorm:"column:question_count" json:"question_count"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
}

func (Tag) TableName() string { return "tags" }

type Question struct {
	ID               uint64    `gorm:"primaryKey;column:id" json:"id"`
	UserID           uint64    `gorm:"column:user_id" json:"user_id"`
	Title            string    `gorm:"column:title" json:"title"`
	Content          string    `gorm:"column:content" json:"content"`
	Status           string    `gorm:"column:status" json:"status"`
	ViewCount        int       `gorm:"column:view_count" json:"view_count"`
	AnswerCount      int       `gorm:"column:answer_count" json:"answer_count"`
	UpvoteCount      int       `gorm:"column:upvote_count" json:"upvote_count"`
	DownvoteCount    int       `gorm:"column:downvote_count" json:"downvote_count"`
	AcceptedAnswerID *uint64   `gorm:"column:accepted_answer_id" json:"accepted_answer_id"`
	LastActivityAt   time.Time `gorm:"column:last_activity_at" json:"last_activity_at"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at" json:"updated_at"`

	Tags []Tag `gorm:"many2many:question_tags;joinForeignKey:QuestionID;joinReferences:TagID" json:"tags"`
}

func (Question) TableName() string { return "questions" }

type Answer struct {
	ID            uint64    `gorm:"primaryKey;column:id" json:"id"`
	QuestionID    uint64    `gorm:"column:question_id" json:"question_id"`
	UserID        uint64    `gorm:"column:user_id" json:"user_id"`
	Content       string    `gorm:"column:content" json:"content"`
	Status        string    `gorm:"column:status" json:"status"`
	UpvoteCount   int       `gorm:"column:upvote_count" json:"upvote_count"`
	DownvoteCount int       `gorm:"column:downvote_count" json:"downvote_count"`
	IsAccepted    bool      `gorm:"column:is_accepted" json:"is_accepted"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Answer) TableName() string { return "answers" }

type Vote struct {
	ID         uint64    `gorm:"primaryKey;column:id" json:"id"`
	UserID     uint64    `gorm:"column:user_id" json:"user_id"`
	TargetType string    `gorm:"column:target_type" json:"target_type"`
	TargetID   uint64    `gorm:"column:target_id" json:"target_id"`
	Value      int       `gorm:"column:value" json:"value"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
}

func (Vote) TableName() string { return "votes" }

type Comment struct {
	ID            uint64    `gorm:"primaryKey;column:id" json:"id"`
	UserID        uint64    `gorm:"column:user_id" json:"user_id"`
	TargetType    string    `gorm:"column:target_type" json:"target_type"`
	TargetID      uint64    `gorm:"column:target_id" json:"target_id"`
	ParentID      uint64    `gorm:"column:parent_id" json:"parent_id"`               // 0 = 根评论；>0 指向根评论 ID
	ReplyToUserID uint64    `gorm:"column:reply_to_user_id" json:"reply_to_user_id"` // 被回复的评论作者
	Content       string    `gorm:"column:content" json:"content"`
	Status        string    `gorm:"column:status" json:"status"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
}

func (Comment) TableName() string { return "comments" }

// 通知类型：回复了你的评论
const NotifCommentReply = "comment_reply"

// Author 内嵌在问题/回答/评论视图中的作者摘要。
type Author struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role"`
}

// Notification 通知类型常量。
const (
	NotifAnswer          = "answer"           // 回答了你的问题
	NotifCommentQuestion = "comment_question" // 评论了你的问题
	NotifCommentAnswer   = "comment_answer"   // 评论了你的回答
	NotifAccept          = "accept"           // 采纳了你的回答
	NotifVoteUpQuestion  = "vote_up_question" // 赞了你的问题
	NotifVoteUpAnswer    = "vote_up_answer"   // 赞了你的回答
)

type Notification struct {
	ID         uint64    `gorm:"primaryKey;column:id" json:"id"`
	UserID     uint64    `gorm:"column:user_id" json:"user_id"`
	Type       string    `gorm:"column:type" json:"type"`
	ActorID    uint64    `gorm:"column:actor_id" json:"actor_id"`
	TargetType string    `gorm:"column:target_type" json:"target_type"`
	TargetID   uint64    `gorm:"column:target_id" json:"target_id"`
	QuestionID uint64    `gorm:"column:question_id" json:"question_id"`
	Content    string    `gorm:"column:content" json:"content"`
	IsRead     bool      `gorm:"column:is_read" json:"is_read"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }

// NotificationView 列表视图：联出触发人昵称与问题标题。
type NotificationView struct {
	Notification
	ActorName     string `json:"actor_name"`
	QuestionTitle string `json:"question_title"`
}
