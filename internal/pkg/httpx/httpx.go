// Package httpx 提供统一的 JSON 响应格式、错误输出与分页参数解析。
package httpx

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/leo0315code/goask/internal/pkg/apperr"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK 输出成功响应；data 为 nil 时输出 null。
func OK(c *gin.Context, data any) {
	c.JSON(200, Response{Code: 0, Message: "ok", Data: data})
}

// Fail 输出错误响应；能识别 apperr.Error 时按其状态码/错误码输出，
// 否则一律按 50000 内部错误处理，不向客户端泄露内部细节。
func Fail(c *gin.Context, err error) {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		c.AbortWithStatusJSON(ae.HTTPStatus, Response{Code: ae.Code, Message: ae.Message})
		return
	}
	c.AbortWithStatusJSON(500, Response{Code: 50000, Message: "服务器内部错误"})
}

// Bind 解析 JSON 请求体，失败时输出 40000 并返回 false。
func Bind(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		Fail(c, apperr.With(apperr.ErrBadRequest, "参数错误: "+err.Error()))
		return false
	}
	return true
}

// PageQuery 通用分页参数，page 从 1 开始，page_size 上限 100。
type PageQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

func (p *PageQuery) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
}

func (p *PageQuery) Offset() int { return (p.Page - 1) * p.PageSize }

// PageData 通用分页响应体。
type PageData struct {
	List     any   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}
