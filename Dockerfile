FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o seed ./cmd/seed

FROM alpine:3.19
WORKDIR /app
# Install netcat for health checks in entrypoint script
RUN apk add --no-cache netcat-openbsd
COPY --from=build /app/server /app/server
COPY --from=build /app/seed /app/seed
COPY scripts/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh
EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]

