# Build from the smartcourse/ directory (the shared lib lives outside this folder):
#   docker build -f services/user-service/Dockerfile -t user-service .
FROM golang:1.26-alpine AS builder

# Mirror the repo layout so "replace smartcourse/libs/common => ../../libs/common" resolves.
WORKDIR /app
COPY libs/common/ ./libs/common/

WORKDIR /app/services/user-service
COPY services/user-service/go.mod services/user-service/go.sum ./
RUN go mod download

COPY services/user-service/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o user-service ./cmd

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/services/user-service/user-service .

EXPOSE 8081
CMD ["./user-service"]
