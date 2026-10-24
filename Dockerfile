# ---- 前端构建（Vue 3 + Vite）----
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# ---- Go 构建 ----
FROM golang:1.27-alpine AS build
WORKDIR /src

# 先拷贝依赖清单，利用 Docker 层缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# 用刚构建的前端产物覆盖，保证镜像内 UI 最新
COPY --from=web /web/dist /src/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/goask ./cmd/server

# ---- 运行阶段 ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -u 10001 goask

WORKDIR /app
COPY --from=build /out/goask /app/goask
COPY migrations /app/migrations

USER goask
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=3 \
    CMD wget -qO- http://127.0.0.1:${PORT:-8080}/healthz || exit 1

ENTRYPOINT ["/app/goask"]
