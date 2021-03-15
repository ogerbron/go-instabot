.PHONY: deps
deps:
	go mod download

.PHONY: lint
lint:
	$(info Running Go code checkers and linters)
	golangci-lint run ./...

.PHONY: test
test:
	go test -race ./...

.PHONY: test-cover
test-cover:
	go test -v -race ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out

.PHONY: go-build
go-build:
	CGO_ENABLED=0 go build -ldflags "-X $(MODULE)/cmd.Version=$$VERSION"
