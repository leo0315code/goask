package repository

import (
	"context"
	"errors"

	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"github.com/leo0315code/goask/internal/pkg/apperr"
)

// Store 持有数据库连接，提供 Repos 构造与事务入口。
type Store struct {
	DB *gorm.DB
}

func NewStore(db *gorm.DB) *Store { return &Store{DB: db} }

// Repos 绑定到同一 DB 执行体（连接或事务）的仓储集合。
type Repos struct {
	Users         *UserRepo
	Questions     *QuestionRepo
	Answers       *AnswerRepo
	Votes         *VoteRepo
	Comments      *CommentRepo
	Tags          *TagRepo
	Spam          *SpamRepo
	Notifications *NotificationRepo
	Admin         *AdminRepo
	Favorites     *FavoriteRepo
}

func NewRepos(db *gorm.DB) *Repos {
	return &Repos{
		Users:         &UserRepo{db: db},
		Questions:     &QuestionRepo{db: db},
		Answers:       &AnswerRepo{db: db},
		Votes:         &VoteRepo{db: db},
		Comments:      &CommentRepo{db: db},
		Tags:          &TagRepo{db: db},
		Spam:          &SpamRepo{db: db},
		Notifications: &NotificationRepo{db: db},
		Admin:         &AdminRepo{db: db},
		Favorites:     &FavoriteRepo{db: db},
	}
}

// Tx 在单个数据库事务中执行 fn；fn 返回 error 时整体回滚。
// fn 内只使用入参 Repos 操作数据，保证所有一致性写入在同一事务内。
func (s *Store) Tx(ctx context.Context, fn func(r *Repos) error) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(NewRepos(tx))
	})
}

// translateNotFound 把 gorm 的记录不存在转换为统一的 404 业务错误。
func translateNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperr.ErrNotFound
	}
	return err
}

// translateDuplicate 把唯一键冲突（MySQL 1062）转换为统一的 409 业务错误。
func translateDuplicate(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperr.ErrConflict
	}
	var me *gomysql.MySQLError
	if errors.As(err, &me) && me.Number == 1062 {
		return apperr.ErrConflict
	}
	return err
}
