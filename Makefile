PROJECT := envsetter
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo development)

.PHONY: build test fmt vet check

commit: fmt
	@govulncheck ./...
	@git add .
	@git commit -a -m "$(filter-out $@,$(MAKECMDGOALS))"
	@git pull
	@git push

%:
	@:

build:
	mkdir -p bin
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/$(PROJECT) ./cmd/envsetter

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

vet:
	go vet ./...

check: test vet
	git diff --check
