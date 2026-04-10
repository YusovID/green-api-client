FROM golang:1.26.1-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.23

RUN adduser -D appuser
USER appuser

WORKDIR /home/appuser

COPY --from=builder --chown=appuser:appuser /server .

EXPOSE 8080

ENTRYPOINT ["./server"]
