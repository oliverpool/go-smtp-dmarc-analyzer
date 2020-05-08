test:
	go test -race ./...

build:
	CGO_ENABLED=0 cd cmd/server && go build -a -o /build/app
