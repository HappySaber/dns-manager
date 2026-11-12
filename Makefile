.PHONY: proto
build:
	go build -o dns-cli cmd/cli/main.go

server-a:
	go run ./cmd/server/main.go --config config/node-a.yaml

server-b:
	go run ./cmd/server/main.go --config config/node-b.yaml

server: 
	go run ./cmd/server/main.go --config config/config.yaml
