FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /dbas ./dbas/cmd/dbas

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /dbas .
COPY dbas/skills/ ./skills/
COPY dbas/frontend/dist/ ./frontend/dist/
COPY dbas/dbas.example.toml ./dbas.example.toml
EXPOSE 8080
ENTRYPOINT ["/app/dbas"]
CMD ["--port", "8080"]
