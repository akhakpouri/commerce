# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Copy workspace and module files for dependency caching
COPY go.work go.work.sum* ./
COPY api/go.mod api/go.sum ./api/
# utils/go.mod required — go.work references ./utils and go mod download
# validates all workspace members. Only go.mod needed, not source or go.sum.
COPY utils/go.mod ./utils/
COPY internal/shared/go.mod internal/shared/go.sum ./internal/shared/

RUN go mod download

# Copy source — only api and internal/shared (utils source not needed)
COPY api/ ./api/
COPY internal/shared/ ./internal/shared/

RUN cd api && go build -o /app/api .

# Runtime stage
FROM alpine:latest

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /app/api .

EXPOSE 8080

USER appuser
CMD ["./api"]
