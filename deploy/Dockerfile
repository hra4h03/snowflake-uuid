FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /snowflake-server ./cmd/snowflake-server

FROM gcr.io/distroless/static-debian12

COPY --from=builder /snowflake-server /snowflake-server

EXPOSE 8080

ENTRYPOINT ["/snowflake-server"]
