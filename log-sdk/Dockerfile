FROM mcr.microsoft.com/oss/go/microsoft/golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o log-sdk-server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/log-sdk-server .

EXPOSE 8080

CMD ["./log-sdk-server"]
