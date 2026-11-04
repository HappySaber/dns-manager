.PHONY: proto
proto:
	mkdir -p gen
	protoc \
		--proto_path=api/proto \
		--go_out=gen --go_opt=paths=source_relative \
		--go-grpc_out=gen --go-grpc_opt=paths=source_relative \
		api/proto/dns/v1/dns.proto