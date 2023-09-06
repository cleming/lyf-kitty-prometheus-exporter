FROM golang:1.19 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o lyf-kitty-exporter

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/lyf-kitty-exporter /usr/local/bin/

EXPOSE 8080

CMD ["lyf-kitty-exporter"]