# Build from the smartcourse/ directory (the shared lib lives outside this folder):
#   docker build -f services/course-service/Dockerfile -t course-service .
FROM golang:1.26-alpine AS builder

# Mirror the repo layout so "replace smartcourse/libs/common => ../../libs/common" resolves.
WORKDIR /app
COPY libs/common/ ./libs/common/

WORKDIR /app/services/course-service
COPY services/course-service/go.mod services/course-service/go.sum ./
RUN go mod download

COPY services/course-service/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o course-service ./cmd

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/services/course-service/course-service .

EXPOSE 8082
CMD ["./course-service"]
