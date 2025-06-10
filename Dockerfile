FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o stresspulse .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /app/stresspulse .
RUN chown -R appuser:appuser /app
USER appuser
ENV TZ=UTC
EXPOSE 9090
ENTRYPOINT ["/app/stresspulse"]
CMD ["-cpu", "50", "-drift", "20", "-pattern", "sine"] 