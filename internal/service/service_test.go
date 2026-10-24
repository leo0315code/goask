package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leo0315code/goask/internal/config"
	"github.com/leo0315code/goask/internal/model"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/captcha"
	"github.com/leo0315code/goask/internal/repository"
)

// ---- 假仓储：仅实现被测路径用到的方法 ----

type fakeUserRepo struct {
	users     map[uint64]*model.User
	createErr error
}

func (f *fakeUserRepo) Create(_ context.Context, u *model.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.users == nil {
		f.users = map[uint64]*model.User{}
	}
	for _, existing := range f.users {
		if existing.Username == u.Username || existing.Email == u.Email {
			return apperr.ErrConflict
		}
	}
	u.ID = uint64(len(f.users) + 1)
	f.users[u.ID] = u
	return nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, id uint64) (*model.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, apperr.ErrNotFound
}

func (f *fakeUserRepo) GetByUsername(_ context.Context, username string) (*model.User, error) {
	for _, u := range f.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, apperr.ErrNotFound
}

func (f *fakeUserRepo) GetByEmailOrUsername(_ context.Context, account string) (*model.User, error) {
	for _, u := range f.users {
		if u.Username == account || u.Email == account {
			return u, nil
		}
	}
	return nil, apperr.ErrNotFound
}

func (f *fakeUserRepo) UpdateProfile(_ context.Context, _ uint64, _, _ string) error { return nil }

func (f *fakeUserRepo) GetByIDs(_ context.Context, ids []uint64) ([]model.User, error) {
	var out []model.User
	for _, id := range ids {
		if u, ok := f.users[id]; ok {
			out = append(out, *u)
		}
	}
	return out, nil
}

// fakeSpam 无操作的防灌水检查。
type fakeSpam struct{}

func (fakeSpam) Check(context.Context, uint64, string) error                  { return nil }
func (fakeSpam) CheckDuplicate(context.Context, uint64, string, string) error { return nil }

func newTestConfig() *config.Config {
	return &config.Config{
		JWTSecret:       "unit-test-secret-32-characters!!",
		JWTExpire:       3600,
		BcryptCost:      4,
		RegisterEnabled: true,
	}
}

// solveCaptcha 解析 "7 + 5 = ?" 形式的算术题。
func solveCaptcha(question string) string {
	var a, b int
	if _, err := fmt.Sscanf(question, "%d + %d", &a, &b); err == nil {
		return fmt.Sprintf("%d", a+b)
	}
	if _, err := fmt.Sscanf(question, "%d - %d", &a, &b); err == nil {
		return fmt.Sprintf("%d", a-b)
	}
	return "0"
}

func TestAuthServiceRegisterAndLogin(t *testing.T) {
	store := captcha.New(time.Minute)
	svc := NewAuthService(&fakeUserRepo{}, newTestConfig(), store)
	ctx := context.Background()

	// 正确流程：先取验证码，再携带答案注册
	id, question := store.New()
	res, err := svc.Register(ctx, RegisterInput{
		Username: "leo", Email: "leo@example.com", Password: "secret123",
		CaptchaID: id, CaptchaCode: solveCaptcha(question),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Token)
	assert.Equal(t, "leo", res.User.Username)
	assert.Equal(t, model.RoleUser, res.User.Role)
	// JSON 序列化不泄露密码哈希（模型 json:"-" 标签）
	b, _ := json.Marshal(res.User)
	assert.NotContains(t, string(b), "password_hash")

	// 验证码错误
	_, err = svc.Register(ctx, RegisterInput{Username: "x1", Email: "x1@example.com", Password: "secret123", CaptchaID: "bad", CaptchaCode: "1"})
	require.Error(t, err)
	assert.Equal(t, 42200, err.(*apperr.Error).Code)

	// 用户名重复（重新取验证码）
	id, question = store.New()
	_, err = svc.Register(ctx, RegisterInput{
		Username: "leo", Email: "other@example.com", Password: "secret123",
		CaptchaID: id, CaptchaCode: solveCaptcha(question),
	})
	require.Error(t, err)
	assert.Equal(t, 40900, err.(*apperr.Error).Code)

	// 用户名格式非法
	id, question = store.New()
	_, err = svc.Register(ctx, RegisterInput{
		Username: "ab", Email: "a@example.com", Password: "secret123",
		CaptchaID: id, CaptchaCode: solveCaptcha(question),
	})
	require.Error(t, err)
	assert.Equal(t, 42200, err.(*apperr.Error).Code)

	// 密码过短
	id, question = store.New()
	_, err = svc.Register(ctx, RegisterInput{
		Username: "leo2", Email: "a@example.com", Password: "short",
		CaptchaID: id, CaptchaCode: solveCaptcha(question),
	})
	require.Error(t, err)
	assert.Equal(t, 42200, err.(*apperr.Error).Code)

	// 用用户名登录
	res, err = svc.Login(ctx, "leo", "secret123")
	require.NoError(t, err)
	assert.NotEmpty(t, res.Token)

	// 用邮箱登录
	_, err = svc.Login(ctx, "leo@example.com", "secret123")
	require.NoError(t, err)

	// 密码错误
	_, err = svc.Login(ctx, "leo", "wrong-pass")
	require.Error(t, err)
	assert.Equal(t, 42200, err.(*apperr.Error).Code)
}

func TestRegisterInviteCodeAndIPLimit(t *testing.T) {
	store := captcha.New(time.Minute)
	cfg := newTestConfig()
	cfg.RegisterInviteCode = "LETMEIN"
	cfg.RegisterMaxPerIPPerDay = 2
	svc := NewAuthService(&fakeUserRepo{}, cfg, store)
	ctx := context.Background()

	reg := func(name, invite, ip string) error {
		id, question := store.New()
		_, err := svc.Register(ctx, RegisterInput{
			Username: name, Email: name + "@example.com", Password: "secret123",
			CaptchaID: id, CaptchaCode: solveCaptcha(question), InviteCode: invite, IP: ip,
		})
		return err
	}

	// 邀请码错误
	err := reg("usr1", "WRONG", "1.1.1.1")
	require.Error(t, err)
	assert.Equal(t, 40300, err.(*apperr.Error).Code)

	// 正确邀请码
	require.NoError(t, reg("usr1", "LETMEIN", "1.1.1.1"))
	require.NoError(t, reg("usr2", "LETMEIN", "1.1.1.1"))

	// 同 IP 第 3 次注册被限制
	err = reg("usr3", "LETMEIN", "1.1.1.1")
	require.Error(t, err)
	assert.Equal(t, 42900, err.(*apperr.Error).Code)

	// 换 IP 不受影响
	require.NoError(t, reg("usr3", "LETMEIN", "2.2.2.2"))

	// 关闭注册
	cfg2 := newTestConfig()
	cfg2.RegisterEnabled = false
	svc2 := NewAuthService(&fakeUserRepo{}, cfg2, captcha.New(time.Minute))
	id, question := svc2.Captcha().New()
	_, err = svc2.Register(ctx, RegisterInput{Username: "usr9", Email: "usr9@example.com", Password: "secret123", CaptchaID: id, CaptchaCode: solveCaptcha(question)})
	require.Error(t, err)
	assert.Equal(t, 40300, err.(*apperr.Error).Code)
}

func TestAuthServiceBannedUserCannotLogin(t *testing.T) {
	repo := &fakeUserRepo{}
	svc := NewAuthService(repo, newTestConfig(), captcha.New(time.Minute))
	ctx := context.Background()

	id, question := svc.Captcha().New()
	res, err := svc.Register(ctx, RegisterInput{
		Username: "bad", Email: "bad@example.com", Password: "secret123",
		CaptchaID: id, CaptchaCode: solveCaptcha(question),
	})
	require.NoError(t, err)
	repo.users[res.User.ID].Status = model.UserStatusBanned

	_, err = svc.Login(ctx, "bad", "secret123")
	require.Error(t, err)
	assert.Equal(t, 40300, err.(*apperr.Error).Code)
}

// ---- 投票服务参数校验 ----

type fakeVoteRepo struct {
	state  *repository.VoteState
	called struct {
		userID     uint64
		targetType string
		targetID   uint64
		value      int
	}
}

func (f *fakeVoteRepo) Vote(_ context.Context, userID uint64, targetType string, targetID uint64, value int) (*repository.VoteState, error) {
	f.called.userID, f.called.targetType, f.called.targetID, f.called.value = userID, targetType, targetID, value
	return f.state, nil
}

func (f *fakeVoteRepo) GetViewerVote(_ context.Context, _ uint64, _ string, _ uint64) (int, error) {
	return 0, nil
}

func TestVoteServiceValidation(t *testing.T) {
	fake := &fakeVoteRepo{state: &repository.VoteState{Value: 1, UpvoteCount: 1}}
	svc := NewVoteService(fake)
	ctx := context.Background()

	state, err := svc.Vote(ctx, 1, model.TargetAnswer, 42, 1)
	require.NoError(t, err)
	assert.Equal(t, repository.VoteState{Value: 1, UpvoteCount: 1}, *state)
	assert.Equal(t, model.TargetAnswer, fake.called.targetType)
	assert.Equal(t, uint64(42), fake.called.targetID)

	// 非法目标类型
	_, err = svc.Vote(ctx, 1, "comment", 1, 1)
	require.Error(t, err)
	assert.Equal(t, 40000, err.(*apperr.Error).Code)

	// 非法 value
	_, err = svc.Vote(ctx, 1, model.TargetQuestion, 1, 0)
	require.Error(t, err)
	assert.Equal(t, 40000, err.(*apperr.Error).Code)

	// 非法目标 ID
	_, err = svc.Vote(ctx, 1, model.TargetQuestion, 0, 1)
	require.Error(t, err)
	assert.Equal(t, 40000, err.(*apperr.Error).Code)
}

// ---- 采纳回答权限 ----

type fakeQuestionRepoForAccept struct {
	repository.QuestionRepo  // 仅覆盖 GetByID / AcceptAnswer
	question                 *model.Question
	acceptedQID, acceptedAID uint64
}

func (f *fakeQuestionRepoForAccept) GetByID(_ context.Context, id uint64) (*model.Question, error) {
	if f.question != nil && f.question.ID == id {
		return f.question, nil
	}
	return nil, apperr.ErrNotFound
}

func (f *fakeQuestionRepoForAccept) AcceptAnswer(_ context.Context, questionID, answerID uint64) error {
	f.acceptedQID, f.acceptedAID = questionID, answerID
	return nil
}

type fakeAnswerRepoForAccept struct {
	repository.AnswerRepo // 仅覆盖 GetByID
	answer                *model.Answer
}

func (f *fakeAnswerRepoForAccept) GetByID(_ context.Context, id uint64) (*model.Answer, error) {
	if f.answer != nil && f.answer.ID == id {
		return f.answer, nil
	}
	return nil, apperr.ErrNotFound
}

func TestAcceptAnswerPermission(t *testing.T) {
	ctx := context.Background()
	question := &model.Question{ID: 10, UserID: 100, Status: model.QuestionStatusOpen}
	answer := &model.Answer{ID: 20, QuestionID: 10, Status: model.AnswerStatusPublished}

	fq := &fakeQuestionRepoForAccept{question: question}
	fa := &fakeAnswerRepoForAccept{answer: answer}
	svc := NewQuestionService(fq, fa, &fakeUserRepo{}, nil, &fakeVoteRepo{}, fakeSpam{})

	// 非提问者且非管理员 → 403
	err := svc.AcceptAnswer(ctx, 999, 10, 20, false)
	require.Error(t, err)
	assert.Equal(t, 40300, err.(*apperr.Error).Code)

	// 提问者 → 采纳成功
	require.NoError(t, svc.AcceptAnswer(ctx, 100, 10, 20, false))
	assert.Equal(t, uint64(10), fq.acceptedQID)
	assert.Equal(t, uint64(20), fq.acceptedAID)

	// 管理员 → 采纳成功
	require.NoError(t, svc.AcceptAnswer(ctx, 999, 10, 20, true))

	// 回答不属于该问题 → 404
	answer.QuestionID = 11
	err = svc.AcceptAnswer(ctx, 100, 10, 20, false)
	require.Error(t, err)
	assert.Equal(t, 40400, err.(*apperr.Error).Code)
}

// ---- 防灌水规则 ----

type fakeSpamRepo struct {
	last    time.Time
	count   int64
	dup     bool
	lastErr error
}

func (f *fakeSpamRepo) LastPostAt(context.Context, string, uint64) (time.Time, error) {
	return f.last, f.lastErr
}

func (f *fakeSpamRepo) RecentCount(context.Context, string, uint64, time.Duration) (int64, error) {
	return f.count, f.lastErr
}

func (f *fakeSpamRepo) ExistsRecentDuplicate(context.Context, string, string, uint64, string, time.Duration) (bool, error) {
	return f.dup, f.lastErr
}

func TestSpamGuardRules(t *testing.T) {
	ctx := context.Background()

	// 刚刚发过 → 触发最小间隔
	g := NewSpamGuard(&fakeSpamRepo{last: time.Now().Add(-5 * time.Second)})
	err := g.Check(ctx, 1, "question")
	require.Error(t, err)
	assert.Equal(t, 42900, err.(*apperr.Error).Code)

	// 上次发布 2 分钟前 → 通过间隔检查；24h 内 20 条 → 触发数量上限
	g = NewSpamGuard(&fakeSpamRepo{last: time.Now().Add(-2 * time.Minute), count: 20})
	err = g.Check(ctx, 1, "question")
	require.Error(t, err)
	assert.Equal(t, 42900, err.(*apperr.Error).Code)

	// 正常通过
	g = NewSpamGuard(&fakeSpamRepo{last: time.Now().Add(-2 * time.Minute), count: 3})
	require.NoError(t, g.Check(ctx, 1, "question"))

	// 重复内容
	g = NewSpamGuard(&fakeSpamRepo{dup: true})
	err = g.CheckDuplicate(ctx, 1, "answer", "相同内容")
	require.Error(t, err)
	assert.Equal(t, 42200, err.(*apperr.Error).Code)

	// 无重复
	g = NewSpamGuard(&fakeSpamRepo{})
	require.NoError(t, g.CheckDuplicate(ctx, 1, "answer", "新内容"))
}
