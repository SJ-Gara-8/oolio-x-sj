# Multi-stage build: static Go binary, no shell — small attack surface for production.
FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /server /server
# Coupon loader downloads .gz files over HTTPS and writes .idx next to cwd — use a volume or emptyDir in K8s.
WORKDIR /data
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/server"]
