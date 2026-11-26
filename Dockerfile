# Образ AYZON EXAM (этап 7): интерфейс собирается Vite, сервер Go отдаёт его сам (WEB_DIR) — один процесс.
# Сборка из корня: docker build -t ayzon-exam . (в корне — потому что mars.cloud и Railway берут Dockerfile оттуда)
FROM node:24-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.27-alpine AS server
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags=-s -o /ayzon-exam ./cmd/server

# Alpine, а не distroless: wget для проверки здоровья. Время Ташкента в коде — FixedZone, tzdata не нужна
FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -H -u 10001 app
COPY --from=server /ayzon-exam /app/ayzon-exam
COPY --from=web /src/frontend/dist /app/web
USER app
WORKDIR /app
ENV ADDR=:8080 WEB_DIR=/app/web
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=30s --retries=3 \
  CMD wget -qO /dev/null http://127.0.0.1:8080/api/health || exit 1
ENTRYPOINT ["/app/ayzon-exam"]
