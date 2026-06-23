FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal

RUN go build -o /bin/gitflame-backend ./cmd/server \
    && go build -o /bin/agent-worker ./cmd/agent-worker

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /bin/gitflame-backend /app/gitflame-backend
COPY --from=builder /bin/agent-worker /app/agent-worker

EXPOSE 8000
CMD ["/app/gitflame-backend"]
