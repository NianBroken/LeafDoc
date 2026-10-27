# Stage 1: Frontend build
FROM node:26-alpine@sha256:40cd011b55415bc894e45b02fdd5a5dfde056921b7d23a64c74f80497af48797 AS frontend

WORKDIR /app/ui/leafwiki-ui
ARG APP_VERSION

COPY ./ui/leafwiki-ui/package.json ./package.json
COPY ./ui/leafwiki-ui/package-lock.json ./package-lock.json
RUN npm ci --ignore-scripts

COPY ./ui/leafwiki-ui/ ./
COPY ./scripts/sync-error-catalogs.mjs /app/scripts/sync-error-catalogs.mjs
RUN mkdir -p /app/internal/core/localization
RUN VITE_API_URL=/ APP_VERSION=${APP_VERSION} npm run build

# Stage 2: Go backend build
FROM golang:1.27.3-alpine@sha256:f2c872bcee949b7be9c490efd02b48a8a2fbf947cdf533f1c58525ec9d8170fd AS builder

ARG APP_VERSION
ARG GOOS
ARG GOARCH
ARG CGO_ENABLED=0
ARG OUTPUT=leafwiki

ENV GOOS=${GOOS}
ENV GOARCH=${GOARCH}
ENV CGO_ENABLED=${CGO_ENABLED}

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Copy built frontend
COPY --from=frontend /app/ui/leafwiki-ui/dist ./internal/http/dist

RUN go build \
  -ldflags="-s -w -X github.com/perber/wiki/internal/http.EmbedFrontend=true -X github.com/perber/wiki/internal/http.Environment=production -X main.Version=${APP_VERSION}" \
  -o /out/${OUTPUT} ./cmd/leafwiki
