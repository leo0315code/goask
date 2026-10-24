// Package apperr 定义带 HTTP 状态码与业务错误码的错误类型，预置常用哨兵错误。
package apperr

// Error 是业务错误。handler 层通过 errors.As 识别并按其字段输出响应；
// 未识别的错误一律按 50000 内部错误处理，不向客户端泄露细节。
type Error struct {
	HTTPStatus int
	Code       int
	Message    string
}

func (e *Error) Error() string { return e.Message }

func New(httpStatus, code int, message string) *Error {
	return &Error{HTTPStatus: httpStatus, Code: code, Message: message}
}

// With 返回同状态码、同错误码但自定义文案的新错误。
func With(e *Error, message string) *Error {
	return New(e.HTTPStatus, e.Code, message)
}

var (
	ErrBadRequest   = New(400, 40000, "参数错误")
	ErrUnauthorized = New(401, 40100, "未登录或登录已过期")
	ErrForbidden    = New(403, 40300, "无权执行该操作")
	ErrNotFound     = New(404, 40400, "资源不存在")
	ErrConflict     = New(409, 40900, "状态冲突")
	ErrValidation   = New(422, 42200, "业务校验失败")
	ErrTooMany      = New(429, 42900, "请求过于频繁，请稍后再试")
	ErrInternal     = New(500, 50000, "服务器内部错误")
)
