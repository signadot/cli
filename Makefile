.PHONY: build

generate-proto:
	protoc --proto_path=. --go_out=. --go_opt=paths=source_relative \
    	internal/api/common.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
    	internal/api/localctrlapi/local_controller_api.proto
	protoc --proto_path=. --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
    	internal/api/rootctrlapi/root_controller_api.proto

build:
	SIGNADOT_IMAGE_SUFFIX='' goreleaser release --rm-dist --snapshot

release:
	SIGNADOT_IMAGE_SUFFIX='' goreleaser release --rm-dist
