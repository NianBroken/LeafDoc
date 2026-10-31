# 前端构建阶段在 Linux 中读取锁文件安装依赖，并生成后端将要嵌入的页面。
FROM node:26-alpine@sha256:0b36e8c136b94cd4fcf02188228e76c31ad5872eef3fec8cbd2eee500cfd9e80 AS frontend
WORKDIR /app/frontend
COPY ./frontend/package.json ./package.json
COPY ./frontend/package-lock.json ./package-lock.json
RUN --mount=type=cache,id=leafdoc-linux-npm,target=/root/.npm npm ci --ignore-scripts --prefer-offline
COPY ./frontend/ ./
COPY ./other/scripts/ /app/other/scripts/
COPY ./backend/internal/core/serviceports/ports.json /app/backend/internal/core/serviceports/ports.json
RUN mkdir -p /app/backend/internal/core/localization
RUN VITE_API_URL=/ npm run build

# 后端构建阶段接收目标架构和 CGO 参数，操作系统固定为 Linux。
# 此文件只输出构建阶段中的二进制，不包含应用运行镜像和启动入口。
FROM golang:1.27.3-alpine@sha256:f2c872bcee949b7be9c490efd02b48a8a2fbf947cdf533f1c58525ec9d8170fd AS builder
ARG APP_VERSION
ARG GOARCH
ARG CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=${GOARCH}
ENV CGO_ENABLED=${CGO_ENABLED}
WORKDIR /build/backend
COPY ./backend/go.mod ./backend/go.sum ./
RUN go mod download
COPY ./backend/ ./
COPY --from=frontend /app/frontend/dist ./internal/http/dist
# 前端静态输出与 Go 源码一起编译，版本值由 APP_VERSION 构建参数提供。
RUN --mount=type=cache,target=/root/.cache/go-build go build \
  -ldflags="-s -w -X github.com/NianBroken/LeafDoc/internal/http.EmbedFrontend=true -X github.com/NianBroken/LeafDoc/internal/http.Environment=production -X main.Version=${APP_VERSION}" \
  -o /out/leafdoc ./cmd/leafdoc
