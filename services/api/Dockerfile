FROM golang:1.25-alpine AS builder
ARG BINARY=api
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /clarityit-bin ./cmd/${BINARY}

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
ARG BINARY=api
COPY --from=builder /clarityit-bin /usr/local/bin/clarityit-bin
EXPOSE 8765
CMD ["/usr/local/bin/clarityit-bin"]
