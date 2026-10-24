package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/leo0315code/goask/internal/pkg/apperr"
	"github.com/leo0315code/goask/internal/pkg/httpx"
)

type UploadHandler struct {
	dir     string
	maxSize int64
}

func NewUploadHandler(dir string, maxSize int64) *UploadHandler {
	return &UploadHandler{dir: dir, maxSize: maxSize}
}

// detectImageExt 通过文件头魔数识别图片类型，返回扩展名；非图片返回 ""。
func detectImageExt(head []byte) string {
	switch {
	case len(head) >= 8 && head[0] == 0x89 && head[1] == 'P' && head[2] == 'N' && head[3] == 'G':
		return ".png"
	case len(head) >= 3 && head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF:
		return ".jpg"
	case len(head) >= 6 && head[0] == 'G' && head[1] == 'I' && head[2] == 'F':
		return ".gif"
	case len(head) >= 12 && string(head[0:4]) == "RIFF" && string(head[8:12]) == "WEBP":
		return ".webp"
	}
	return ""
}

// Image 上传图片（需登录）：multipart 字段 file。
// 校验魔数与大小，随机文件名防覆盖，按年月分目录存储；返回可访问 URL。
func (h *UploadHandler) Image(c *gin.Context) {
	if _, ok := currentUserID(c); !ok {
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		httpx.Fail(c, apperr.With(apperr.ErrBadRequest, "缺少文件字段 file"))
		return
	}
	if fh.Size > h.maxSize {
		httpx.Fail(c, apperr.With(apperr.ErrBadRequest, fmt.Sprintf("图片不能超过 %dMB", h.maxSize/1024/1024)))
		return
	}

	src, err := fh.Open()
	if err != nil {
		httpx.Fail(c, apperr.ErrInternal)
		return
	}
	defer src.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	ext := detectImageExt(head[:n])
	if ext == "" {
		httpx.Fail(c, apperr.With(apperr.ErrBadRequest, "仅支持 png / jpg / gif / webp 图片"))
		return
	}

	subDir := time.Now().Format("2006/01")
	dirPath := filepath.Join(h.dir, subDir)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		httpx.Fail(c, apperr.ErrInternal)
		return
	}

	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	name := hex.EncodeToString(buf) + ext
	dst, err := os.Create(filepath.Join(dirPath, name))
	if err != nil {
		httpx.Fail(c, apperr.ErrInternal)
		return
	}
	defer dst.Close()

	// 魔数已消费，回滚到文件头再完整落盘
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		httpx.Fail(c, apperr.ErrInternal)
		return
	}
	if _, err := io.Copy(dst, src); err != nil {
		httpx.Fail(c, apperr.ErrInternal)
		return
	}

	urlPath := "/uploads/" + filepath.ToSlash(filepath.Join(subDir, name))
	if uid, ok := currentUserID(c); ok {
		slog.Info("image uploaded", "user_id", uid, "size", fh.Size, "path", urlPath)
	}
	httpx.OK(c, gin.H{"url": urlPath})
}
