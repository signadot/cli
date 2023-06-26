.PHONY: build

generate-proto:
	protoc --proto_path=. --go_out=. --go_opt=paths=source_relative \
    	internal/locald/api/common.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
    	internal/locald/api/sandboxmanager/sandbox_manager_api.proto
	protoc --proto_path=. --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
    	internal/locald/api/localnet/localnet_api.proto

build:
	SIGNADOT_IMAGE_SUFFIX='' goreleaser release --rm-dist --snapshot

release:
	SIGNADOT_IMAGE_SUFFIX='' goreleaser release --rm-dist
