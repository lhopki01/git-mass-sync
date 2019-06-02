test:
	go test ./...

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out

lint:
	golangci-lint run

release:
	git tag -a $$VERSION
	git push origin $$VERSION
	goreleaser
