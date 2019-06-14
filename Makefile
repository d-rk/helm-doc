BUILD_TS := $(shell date -Iseconds --utc)
COMMIT_SHA := $(shell git rev-parse HEAD)
VERSION := $(shell git describe --abbrev=0 --tags)

export CGO_ENABLED=0
export GOOS=linux
export GO111MODULE=on

project=github.com/random-dwi/helm-doc
ld_flags := "-X $(project)/cmd.version=$(VERSION) -X $(project)/cmd.gitCommit=$(COMMIT_SHA) -X $(project)/cmd.buildTime=$(BUILD_TS)"

.DEFAULT_GOAL := all
.PHONY: all
all: vet test build

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	go build -ldflags $(ld_flags) -o helm-doc

.PHONY: clean
clean:
	rm -f helm-doc
	go clean -testcache

# usage make version=0.0.4 release
#
# manually executing goreleaser:
# export GITHUB_TOKEN=xyz
# goreleaser --rm-dist
#
.PHONY: release
release:
	eval "sed -i 's/version:.*/version: \"$$version\"/g' plugin.yaml"
	git add "plugin.yaml"
	git commit -m "releases $(version)"
	git tag -a $(version) -m "release $(version)"
	git push origin
	git push origin $(version)
