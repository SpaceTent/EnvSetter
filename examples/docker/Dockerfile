# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS envsetter-build
WORKDIR /src
RUN --mount=type=cache,target=/go/pkg/mod \
    go install github.com/SpaceTent/EnvSetter/cmd/envsetter@latest

FROM golang:1.24-alpine AS app-build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/app ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -H -u 10001 app
COPY --from=envsetter-build /go/bin/envsetter /usr/local/bin/envsetter
COPY --from=app-build /out/app /usr/local/bin/app
USER app
ENTRYPOINT ["envsetter", "run", "--", "/usr/local/bin/app"]
