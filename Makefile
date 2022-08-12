.PHONY: build

build:
	SIGNADOT_IMAGE_SUFFIX='' goreleaser release --rm-dist --snapshot

release:
	SIGNADOT_IMAGE_SUFFIX='' goreleaser release --rm-dist
