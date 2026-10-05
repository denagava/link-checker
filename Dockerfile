FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /link-checker .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 1000 app
USER app
WORKDIR /data
COPY --from=builder /link-checker /usr/local/bin/link-checker
EXPOSE 8080
ENTRYPOINT ["link-checker", "-addr", ":8080", "-data", "/data/links_data.json"]
