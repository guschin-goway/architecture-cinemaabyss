FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy && go build -o proxy main.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/proxy .
CMD ["./proxy"]
