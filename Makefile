PROJECT := envsetter
VERSION := $(shell git describe --tag --abbrev=0)
NEXT_VERSION:=$(shell git describe --tags --abbrev=0 | awk -F . '{OFS="."; $$NF+=1; print}')
SHA1 := $(shell git rev-parse HEAD)
NOW := $(shell date -u +'%Y%m%d-%H%M%S')

tag:
	@echo "$(VERSION)"

search:
	@echo "Searching for $(s)"
	@grep -r -n --exclude-dir=static "$(s)" *

build:
	go generate
	go build -o bin/$(PROJECT) -ldflags "-X main.VERSION=$(VERSION)" .
	go build -o bin/envsetter ./cmd/envsetter


release: fmt
	@git tag -a $(NEXT_VERSION) -m "Release $(NEXT_VERSION)"
	@git push --all
	@git push --tags

newfeature:
	@git checkout -b "Feature-$(NOW)"

reset:
	@git checkout master
	@git pull
	@git fetch
	@git reset --hard origin/master

fmt:
	@go mod tidy
	@goimports -w .
	@gofmt -w -s .
	@go clean ./...


commit: fmt
	@govulncheck ./...
	@git add .
	@git commit -a -m "$(filter-out $@,$(MAKECMDGOALS))"
	@git pull
	@git push

%:
	@:

update:
	@go get -u ./...
	@go mod tidy

test:
	@go test -v -coverprofile=profile.cov ./...
	@go tool cover -func profile.cov


run:
	@go run .

lint:
	@go fmt ./...
	@go vet ./...
	@goimports -w .
	@echo "fmt vet imports done."
	@golangci-lint run ./...
	@echo "Lint Complete"
