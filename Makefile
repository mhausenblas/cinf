release_version:= v0.5.0

export GO111MODULE=on

.PHONY: bin
bin:
	go build -o bin/cinf github.com/mhausenblas/cinf

.PHONY: release
release:
	curl -sL https://git.io/goreleaser | bash -s -- --rm-dist --config .goreleaser.yml

.PHONY: publish
publish:
	git tag ${release_version}
	git push --tags