# Step 1: Frontend
FROM node:26-alpine@sha256:0b36e8c136b94cd4fcf02188228e76c31ad5872eef3fec8cbd2eee500cfd9e80 AS frontend-build
WORKDIR /app/frontend
COPY ./frontend/package*.json ./
RUN --mount=type=cache,id=leafdoc-linux-npm,target=/root/.npm npm ci --ignore-scripts --prefer-offline
COPY ./frontend/ ./
COPY ./other/scripts/ /app/other/scripts/
COPY ./backend/internal/core/serviceports/ports.json /app/backend/internal/core/serviceports/ports.json
RUN mkdir -p /app/backend/internal/core/localization
RUN VITE_API_URL=/ npm run build

# Step 2: Backend + Build binary
FROM golang:1.27.3-alpine@sha256:f2c872bcee949b7be9c490efd02b48a8a2fbf947cdf533f1c58525ec9d8170fd AS backend-build
WORKDIR /app/backend
ARG APP_VERSION
COPY ./backend/go.mod ./backend/go.sum ./
RUN go mod download
COPY ./backend/ ./
COPY --from=frontend-build /app/frontend/dist ./internal/http/dist
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/NianBroken/LeafDoc/internal/http.EmbedFrontend=true -X github.com/NianBroken/LeafDoc/internal/http.Environment=production -X main.Version=${APP_VERSION}" \
    -o /out/leafdoc ./cmd/leafdoc

# Step 3: Final image (small)
FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS final
RUN apk add --no-cache bash lsof procps ca-certificates
WORKDIR /app
COPY --from=backend-build /out/leafdoc /app/leafdoc
COPY ./other/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh
EXPOSE 24692
RUN mkdir -p /app/data && chmod 777 /app/data
LABEL org.opencontainers.image.title="LeafDoc" \
      org.opencontainers.image.description="LeafDoc – A fast wiki for people who think in folders, not feeds" \
      org.opencontainers.image.url="https://github.com/NianBroken/LeafDoc" \
      org.opencontainers.image.licenses="MIT"
ENTRYPOINT ["/docker-entrypoint.sh"]
