.PHONY: build

build:
	goreleaser release --rm-dist --snapshot

release:
	goreleaser release --rm-dist
