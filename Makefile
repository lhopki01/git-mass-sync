MODULE := github.com/lhopki01/git-mass-sync

test:
	go test -race ./...

test-cover:
	go test -race ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out

lint:
	golangci-lint run

release:
	git tag -a $$VERSION
	git push origin $$VERSION
	goreleaser --rm-dist

build:
	CGO_ENABLED=0 go build -ldflags "-X $(MODULE)/cmd.Version=$$VERSION"
