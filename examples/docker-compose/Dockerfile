FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /envsetter ./cmd/envsetter

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -H -u 10001 app
COPY --from=build /envsetter /usr/local/bin/envsetter
USER app
ENTRYPOINT ["envsetter", "run", "--token-file", "/run/secrets/envvault_token", "--key-file", "/run/secrets/envvault_key", "--"]
CMD ["sh", "-c", "echo application started; exec sleep 3600"]
