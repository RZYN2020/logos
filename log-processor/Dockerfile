FROM mcr.microsoft.com/oss/go/microsoft/golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod tidy

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o log-processor ./cmd/job

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/log-processor .

EXPOSE 9090

CMD ["./log-processor"]