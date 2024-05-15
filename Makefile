.PHONY: check-tag check-token publish tag tidy

TAG = ${NEW_RELEASE_TAG}
TARGET = novelist

check-tag:
ifndef NEW_RELEASE_TAG
	$(error Please set the NEW_RELEASE_TAG env variable)
	exit 1
endif

check-token:
ifndef GITHUB_TOKEN
	$(error Please set the GITHUB_TOKEN env variable)
	exit 1
endif

publish: tidy tag check-token
	go install github.com/goreleaser/goreleaser@latest
	goreleaser release --clean

tag: check-tag
	git tag -a "$(TAG)" -m "$(TAG)"
	git push origin $(TAG)

tidy:
	go mod tidy