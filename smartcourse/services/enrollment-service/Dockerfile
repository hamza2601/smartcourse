# Build from the smartcourse/ directory (the shared lib lives outside this folder):
#   docker build -f services/enrollment-service/Dockerfile -t enrollment-service .
FROM golang:1.26-alpine AS builder

# Mirror the repo layout so "replace smartcourse/libs/common => ../../libs/common" resolves.
WORKDIR /app
COPY libs/common/ ./libs/common/

WORKDIR /app/services/enrollment-service
COPY services/enrollment-service/go.mod services/enrollment-service/go.sum ./
RUN go mod download

COPY services/enrollment-service/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o enrollment-service ./cmd

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/services/enrollment-service/enrollment-service .

EXPOSE 8083
CMD ["./enrollment-service"]
