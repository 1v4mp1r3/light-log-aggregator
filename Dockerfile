FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/loglite ./cmd/loglite

FROM alpine:3.22
RUN adduser -D -H -u 10001 loglite && mkdir -p /data && chown -R loglite:loglite /data
USER loglite
COPY --from=build /out/loglite /usr/local/bin/loglite
EXPOSE 8080 5514/udp
ENTRYPOINT ["loglite"]
