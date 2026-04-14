# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Copy workspace and module files for dependency caching
COPY go.work go.work.sum* ./
COPY api/go.mod api/go.sum ./api/
COPY utils/go.mod utils/go.sum* ./utils/
COPY internal/shared/go.mod internal/shared/go.sum ./internal/shared/

# Stub out utils so go.work resolves without copying its source
RUN mkdir -p utils && test -f utils/go.mod || echo 'module commerce/utils' > utils/go.mod

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
