FROM golang:1.25-alpine AS builder
ARG BINARY=api
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /clarityit-bin ./cmd/${BINARY}

FROM alpine:3.21
RUN apk --no-cache add ca-certificates && \
    addgroup -S clarityit && adduser -S clarityit -G clarityit
ARG BINARY=api
COPY --from=builder /clarityit-bin /usr/local/bin/clarityit-bin
USER clarityit
EXPOSE 8765
# Healthcheck only meaningful for the API binary
# Workers don't serve HTTP — their health is tracked via deep health
CMD ["/usr/local/bin/clarityit-bin"]
