# 此构建文件位于 installers/，构建上下文必须是仓库根目录。
# 前端、后端与运行阶段沿用项目正式镜像的固定 Linux 工具链和源码入口。
FROM node:26-alpine@sha256:0b36e8c136b94cd4fcf02188228e76c31ad5872eef3fec8cbd2eee500cfd9e80 AS frontend-build
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN --mount=type=cache,id=leafdoc-linux-npm,target=/root/.npm npm ci --ignore-scripts --prefer-offline
COPY frontend/ ./
COPY other/scripts/ /app/other/scripts/
COPY other/project.json /app/other/project.json
COPY backend/internal/core/serviceports/ports.json /app/backend/internal/core/serviceports/ports.json
RUN mkdir -p /app/backend/internal/core/localization
RUN VITE_API_URL=/ npm run build

FROM golang:1.27.3-alpine@sha256:f2c872bcee949b7be9c490efd02b48a8a2fbf947cdf533f1c58525ec9d8170fd AS backend-build
WORKDIR /app/backend
ARG APP_VERSION=local
ARG APP_NAME=LeafDoc
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-build /app/frontend/dist ./internal/http/dist
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/NianBroken/LeafDoc/internal/http.EmbedFrontend=true -X github.com/NianBroken/LeafDoc/internal/http.Environment=production -X main.Version=${APP_VERSION} -X main.ProjectName=${APP_NAME}" \
    -o /out/leafdoc ./cmd/leafdoc

FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
ARG APP_NAME=LeafDoc
ARG APP_VERSION=local
RUN apk add --no-cache bash ca-certificates openssl wget \
    && addgroup -S leafdoc \
    && adduser -S -G leafdoc leafdoc \
    && mkdir -p /app/data \
    && chown leafdoc:leafdoc /app/data \
    && chmod 750 /app/data
WORKDIR /app
COPY --from=backend-build /out/leafdoc /app/leafdoc
COPY other/docker-entrypoint.sh /docker-entrypoint.sh
COPY installers/entrypoint.sh /installer-entrypoint.sh
COPY other/scripts/prepare-instance.sh /prepare-instance.sh
RUN chmod 755 /docker-entrypoint.sh /installer-entrypoint.sh /prepare-instance.sh
USER leafdoc
EXPOSE 24692
VOLUME ["/app/data"]
HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=12 \
    CMD wget -qO- "http://127.0.0.1:24692${LEAFDOC_BASE_PATH:-}/api/health" >/dev/null || exit 1
LABEL org.opencontainers.image.title="${APP_NAME}" \
      org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.url="https://github.com/NianBroken/LeafDoc" \
      org.opencontainers.image.licenses="MIT"
ENTRYPOINT ["/installer-entrypoint.sh"]
