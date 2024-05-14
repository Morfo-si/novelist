.PHONY: check-env publish tag tidy

TAG = ${NEW_RELEASE_TAG}
TARGET = novelist

check-env:
ifndef NEW_RELEASE_TAG
	$(error Please set the NEW_RELEASE_TAG env variable)
	exit 1
endif

publish: tidy tag
	GOARCH=arm64 GOOS=darwin go build -o $(TARGET)-$(TAG)-darwin-arm64
	GOARCH=arm64 GOOS=linux go build -o $(TARGET)-$(TAG)-linux-arm64
	GOARCH=amd64 GOOS=linux go build -o $(TARGET)-$(TAG)-linux-amd64
	GOARCH=amd64 GOOS=windows go build -o $(TARGET)-$(TAG)-windows-amd64.exe

tag: check-env
	git tag -a "$(TAG)" -m "$(TAG)"
	git push origin $(TAG)

tidy: check-env
	go mod tidy