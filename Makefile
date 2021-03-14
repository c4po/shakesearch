
build:
	@go build -o bin/shakesearch

fmt:
	go mod tidy
	gofmt -s -w .
# 	golangci-lint run ./

tools:
	GO111MODULE=on go install github.com/golangci/golangci-lint/cmd/golangci-lint
