# DBAS Docker image
# Multi-stage: build with Go, ship a distroless static binary
#
# Build:
#   docker build -t dbas:latest -f dbas/Dockerfile .
#
# Run (demo mode):
#   docker run -p 8080:8080 dbas:latest --demo --port 8080
#
# Run (with config):
#   docker run -p 8080:8080 -v /path/to/dbas.toml:/etc/dbas/dbas.toml dbas:latest -config /etc/dbas/dbas.toml

FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /dbas ./dbas/cmd/dbas

FROM alpine:3.21
RUN apk add --no-cache ca-certificates bash git
COPY --from=builder /dbas /usr/local/bin/dbas
COPY --from=builder /src/dbas/frontend/dist /frontend/dist
COPY --from=builder /src/dbas/skills /skills
COPY --from=builder /src/dbas/dbas.example.toml /etc/dbas/dbas.example.toml

EXPOSE 8080
VOLUME ["/data"]
WORKDIR /data

ENTRYPOINT ["dbas"]
CMD ["--demo", "--port", "8080"]
