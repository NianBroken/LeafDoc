# Stage 1: Frontend build
FROM node:26-alpine@sha256:0b36e8c136b94cd4fcf02188228e76c31ad5872eef3fec8cbd2eee500cfd9e80 AS frontend

WORKDIR /app/ui

COPY ./ui/package.json ./package.json
COPY ./ui/package-lock.json ./package-lock.json
RUN --mount=type=cache,id=leafdoc-linux-npm,target=/root/.npm npm ci --ignore-scripts --prefer-offline

COPY ./ui/ ./
COPY ./scripts/sync-error-catalogs.mjs /app/scripts/sync-error-catalogs.mjs
COPY ./scripts/resolve-version.mjs /app/scripts/resolve-version.mjs
COPY ./scripts/release-service-ports.mjs ./scripts/release-service-ports.d.mts /app/scripts/
COPY ./internal/core/serviceports/ports.json /app/internal/core/serviceports/ports.json
RUN mkdir -p /app/internal/core/localization
RUN VITE_API_URL=/ npm run build

# Stage 2: Go backend build
FROM golang:1.27.3-alpine@sha256:f2c872bcee949b7be9c490efd02b48a8a2fbf947cdf533f1c58525ec9d8170fd AS builder

ARG APP_VERSION
ARG GOARCH
ARG CGO_ENABLED=0

ENV GOOS=linux
ENV GOARCH=${GOARCH}
ENV CGO_ENABLED=${CGO_ENABLED}

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Copy built frontend
COPY --from=frontend /app/ui/dist ./internal/http/dist

RUN --mount=type=cache,target=/root/.cache/go-build go build \
  -ldflags="-s -w -X github.com/NianBroken/LeafDoc/internal/http.EmbedFrontend=true -X github.com/NianBroken/LeafDoc/internal/http.Environment=production -X main.Version=${APP_VERSION}" \
  -o /out/leafdoc ./cmd/leafdoc
