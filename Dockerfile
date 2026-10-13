ARG GO_IMAGE=golang:1.26-bookworm
FROM ${GO_IMAGE} AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=1 go build -buildvcs=false -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=1 go build -buildvcs=false -trimpath -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=1 go build -buildvcs=false -trimpath -o /out/seed ./cmd/seed && \
    CGO_ENABLED=1 go build -buildvcs=false -trimpath -o /out/check-schema ./cmd/check-schema

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=build /out/ /app/
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN mkdir -p /app/storage/app/public && chown -R 10001:10001 /app
USER 10001:10001
EXPOSE 8000
ENV HTTP_ADDR=:8000
ENTRYPOINT ["/app/api"]
