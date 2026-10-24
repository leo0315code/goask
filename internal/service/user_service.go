package service

import (
	"context"

	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/passwordx"
)

// UserService 用户账号与管理工作。
type UserService struct {
	users      UserRepo
	bcryptCost int
}

func NewUserService(users UserRepo, bcryptCost int) *UserService {
	return &UserService{users: users, bcryptCost: bcryptCost}
}

// ChangePassword 修改密码并使所有旧 token 失效，返回更新后的用户（用于签发新 token）。
func (s *UserService) ChangePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) (*model.User, error) {
	if oldPassword == "" || len(newPassword) < 8 || len(newPassword) > 72 {
		return nil, apperr.With(apperr.ErrValidation, "新密码长度需为 8-72 字节，且原密码不能为空")
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !passwordx.Verify(u.PasswordHash, oldPassword) {
		return nil, apperr.With(apperr.ErrValidation, "原密码错误")
	}
	hash, err := passwordx.Hash(newPassword, s.bcryptCost)
	if err != nil {
		return nil, err
	}
	if err := s.users.UpdatePassword(ctx, userID, hash); err != nil {
		return nil, err
	}
	if err := s.users.BumpTokenVersion(ctx, userID); err != nil {
		return nil, err
	}
	return s.users.GetByID(ctx, userID)
}

// LogoutEverywhere 使所有已签发 token 失效，返回更新后的用户。
func (s *UserService) LogoutEverywhere(ctx context.Context, userID uint64) (*model.User, error) {
	if err := s.users.BumpTokenVersion(ctx, userID); err != nil {
		return nil, err
	}
	return s.users.GetByID(ctx, userID)
}

// ListUsers 管理端用户列表。
func (s *UserService) ListUsers(ctx context.Context, q string, page, pageSize int) ([]model.User, int64, error) {
	return s.users.List(ctx, q, page, pageSize)
}

// SetStatus 封禁 / 解封用户（管理员操作）。
func (s *UserService) SetStatus(ctx context.Context, operator *model.User, targetID uint64, status string) error {
	if status != model.UserStatusActive && status != model.UserStatusBanned {
		return apperr.With(apperr.ErrBadRequest, "status 取值只能为 active 或 banned")
	}
	if targetID == operator.ID {
		return apperr.With(apperr.ErrValidation, "不能对自己执行封禁操作")
	}
	target, err := s.users.GetByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.Role == model.RoleAdmin {
		return apperr.With(apperr.ErrForbidden, "不能封禁管理员账号")
	}
	return s.users.SetStatus(ctx, targetID, status)
}
