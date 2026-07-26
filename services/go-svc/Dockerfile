FROM golang:1.25-alpine AS build
WORKDIR /src/services/go-svc
COPY services/go-svc/go.mod services/go-svc/go.sum ./
RUN go mod download
COPY services/go-svc ./

FROM build AS api-build
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /go-svc-api ./cmd/api

FROM build AS worker-build
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /go-svc-worker ./cmd/worker

FROM alpine:3.20 AS api
WORKDIR /app
COPY --from=api-build /go-svc-api /app/go-svc-api
EXPOSE 8081
CMD ["/app/go-svc-api"]

FROM alpine:3.20 AS worker
WORKDIR /app
COPY --from=worker-build /go-svc-worker /app/go-svc-worker
CMD ["/app/go-svc-worker"]
