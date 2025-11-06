FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o community

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/community /app/community
ENV PORT=7000
CMD ["/app/community"]
