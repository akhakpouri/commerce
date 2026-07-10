# Build stage
FROM golang:1.26.5-alpine AS builder

WORKDIR /build

# Copy workspace and module files for dependency caching
COPY go.work go.work.sum* ./
COPY utils/go.mod utils/go.sum* ./utils/
# api/go.mod required — go.work references ./api and go mod download
# validates all workspace members. Only go.mod needed, not source or go.sum.
COPY api/go.mod ./api/
COPY internal/shared/go.mod internal/shared/go.sum ./internal/shared/

RUN go mod download

# Copy source — only utils and internal/shared (api source not needed)
COPY utils/ ./utils/
COPY internal/shared/ ./internal/shared/

# config.json is gitignored — create a dummy so //go:embed succeeds.
# At runtime, JSON parse fails and NewDbConfig falls back to env vars.
RUN echo '{}' > ./utils/configs/config.json

RUN cd utils && go build -o /app/utils .

# Runtime stage
FROM alpine:latest

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /app/utils .

USER appuser
CMD ["./utils"]
