BINARY_NAME := ts-flatten
BIN_DIR ?= ./bin

.DEFAULT_GOAL := build
.PHONY: build clean test

build:
	go build -ldflags="-s -w" -o "$(BIN_DIR)/$(BINARY_NAME)" .

clean:
	rm -rf "$(BIN_DIR)"

test:
	go test ./...

# Publish the next stable version, matching gh-agent's make tag-* interface.
.PHONY: tag-patch tag-minor tag-major
tag-patch tag-minor tag-major:
	@set -eu; \
	if [ -n "$$(git status --porcelain)" ]; then \
		echo "Commit or stash changes before publishing a version." >&2; \
		exit 1; \
	fi; \
	git fetch origin --tags; \
	latest=$$(git tag --list 'v*' --sort=-version:refname | sed -n '/^v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*$$/{p;q;}'); \
	version=$${latest:-v0.0.0}; \
	version=$${version#v}; \
	major=$${version%%.*}; \
	rest=$${version#*.}; \
	minor=$${rest%%.*}; \
	patch=$${rest##*.}; \
	case "$@" in \
		tag-patch) patch=$$((patch + 1)) ;; \
		tag-minor) minor=$$((minor + 1)); patch=0 ;; \
		tag-major) major=$$((major + 1)); minor=0; patch=0 ;; \
	esac; \
	tag="v$$major.$$minor.$$patch"; \
	git tag "$$tag"; \
	git push origin "refs/tags/$$tag"; \
	printf 'Tagged and pushed %s\n' "$$tag"
