FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o aws-cost .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/aws-cost .
CMD ["./aws-cost"]
