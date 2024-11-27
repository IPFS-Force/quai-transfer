VERSION:=commit_$(shell git rev-parse --short HEAD)_time_$(shell date +"%Y-%m-%dT%H:%M:%S")
BUILDTIME := $(shell date +"%Y-%m-%dT%H:%M:%S")

GOLDFLAGS += -X main.Version=$(VERSION)
GOLDFLAGS += -X main.Buildtime=$(BUILDTIME)
GOFLAGS = -ldflags "$(GOLDFLAGS)"

linux-amd64-api:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app/quai-wallet-linux-amd64 $(GOFLAGS) ./cmd/

linux-arm64-api:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o app/quai-wallet-linux-arm64 $(GOFLAGS) ./cmd/

darwin-arm64-api:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o app/quai-wallet-darwin-arm64 $(GOFLAGS) ./cmd/

darwin-amd64-api:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o app/quai-wallet-darwin-amd64 $(GOFLAGS) ./cmd/

windows-amd64-api:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o app/quai-wallet-windows-amd64 $(GOFLAGS) ./cmd/

build:
	go build $(GOFLAGS) -o app/quai-wallet ./cmd/
