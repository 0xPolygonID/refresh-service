test:
	go test -v -count=1 ./...

lint:
	golangci-lint run
