FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /out/hazmat-validator ./cmd/server
FROM alpine:3.21
RUN addgroup -S hazmat && adduser -S -G hazmat hazmat
WORKDIR /app
COPY --from=build /out/hazmat-validator /usr/local/bin/hazmat-validator
COPY configs/config.yaml /app/configs/config.yaml
USER hazmat
EXPOSE 32710
ENTRYPOINT ["hazmat-validator", "-config", "/app/configs/config.yaml"]
