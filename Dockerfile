FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod go.sum ./
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:latest

WORKDIR /app
RUN mkdir -p /app/config
COPY --from=0 /server .
# COPY config/ /app/config/

EXPOSE 50051

CMD ["./server"]