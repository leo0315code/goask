package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
)

type UserRepo struct {
	db *gorm.DB
}

func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	return translateDuplicate(r.db.WithContext(ctx).Create(u).Error)
}

func (r *UserRepo) GetByID(ctx context.Context, id uint64) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &u, nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &u, nil
}

func (r *UserRepo) GetByEmailOrUsername(ctx context.Context, account string) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).Where("email = ? OR username = ?", account, account).First(&u).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &u, nil
}

func (r *UserRepo) GetByIDs(ctx context.Context, ids []uint64) ([]model.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []model.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepo) UpdateProfile(ctx context.Context, id uint64, bio, avatarURL string) error {
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).
		Updates(map[string]any{"bio": bio, "avatar_url": avatarURL})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id uint64, hash string) error {
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).
		Update("password_hash", hash)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// SetStatus 管理员设置用户状态（active / banned）。
func (r *UserRepo) SetStatus(ctx context.Context, id uint64, status string) error {
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).
		Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// List 管理端用户列表，按用户名/邮箱模糊搜索。
func (r *UserRepo) List(ctx context.Context, q string, page, pageSize int) ([]model.User, int64, error) {
	base := func() *gorm.DB {
		db := r.db.WithContext(ctx).Model(&model.User{})
		if q != "" {
			db = db.Where("username LIKE ? OR email LIKE ?", "%"+q+"%", "%"+q+"%")
		}
		return db
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []model.User
	err := base().Order("id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&users).Error
	return users, total, err
}
