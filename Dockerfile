# Step 1: Frontend
FROM node:26-alpine@sha256:40cd011b55415bc894e45b02fdd5a5dfde056921b7d23a64c74f80497af48797 AS frontend-build
WORKDIR /app/ui/leafwiki-ui
ARG APP_VERSION
COPY ./ui/leafwiki-ui/package*.json ./
RUN npm ci --ignore-scripts
COPY ./ui/leafwiki-ui/ ./
COPY ./scripts/sync-error-catalogs.mjs /app/scripts/sync-error-catalogs.mjs
RUN mkdir -p /app/internal/core/localization
RUN VITE_API_URL=/ APP_VERSION=${APP_VERSION} npm run build

# Step 2: Backend + Build binary
FROM golang:1.27.3-alpine@sha256:f2c872bcee949b7be9c490efd02b48a8a2fbf947cdf533f1c58525ec9d8170fd AS backend-build
WORKDIR /app
ARG APP_VERSION
ARG RATE_LIMIT_BYPASS=false
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-build /app/ui/leafwiki-ui/dist ./internal/http/dist
RUN CGO_ENABLED=0 go build \
	-ldflags="-s -w -X github.com/perber/wiki/internal/http.EmbedFrontend=true -X github.com/perber/wiki/internal/http.Environment=production -X main.Version=${APP_VERSION} -X github.com/perber/wiki/internal/wiki/auth.DisableRefreshTokenRateLimit=${RATE_LIMIT_BYPASS}" \
	-o /out/leafwiki ./cmd/leafwiki

# Step 3: Final image (small)
FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS final
WORKDIR /app
COPY --from=backend-build /out/leafwiki /app/leafwiki

COPY ./docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

EXPOSE 8080

RUN mkdir -p /app/data && chmod 777 /app/data

LABEL org.opencontainers.image.title="LeafWiki" \
      org.opencontainers.image.description="LeafWiki – A fast wiki for people who think in folders, not feeds" \
      org.opencontainers.image.url="https://demo.leafwiki.com" \
      org.opencontainers.image.licenses="MIT"

ENTRYPOINT ["/docker-entrypoint.sh"]
