// Package handler 实现 HTTP 协议层：参数绑定校验、调用 service、统一响应。
// 本文件提供 handler 共用的小工具。
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/middleware"
	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/httpx"
)

// parseID 解析路径参数中的正整数 ID，失败时输出 400 响应。
func parseID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		httpx.Fail(c, apperr.With(apperr.ErrBadRequest, "无效的 ID: "+c.Param(name)))
		return 0, false
	}
	return id, true
}

// currentUserID 取当前登录用户 ID，未登录输出 401。
func currentUserID(c *gin.Context) (uint64, bool) {
	uid, ok := middleware.UserID(c)
	if !ok {
		httpx.Fail(c, apperr.ErrUnauthorized)
		return 0, false
	}
	return uid, true
}

// pageQuery 解析并规范化分页参数。
func pageQuery(c *gin.Context) httpx.PageQuery {
	var pq httpx.PageQuery
	_ = c.ShouldBindQuery(&pq)
	pq.Normalize()
	return pq
}

// pageOK 包装分页数据输出。
func pageOK(c *gin.Context, list any, total int64, pq httpx.PageQuery) {
	httpx.OK(c, httpx.PageData{List: list, Total: total, Page: pq.Page, PageSize: pq.PageSize})
}
