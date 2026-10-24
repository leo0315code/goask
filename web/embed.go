// Package web 内嵌 Web 前端构建产物（web/dist，由 Vite 构建生成），随二进制一起发布。
package web

import "embed"

//go:embed all:dist
var Dist embed.FS
