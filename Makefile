NAME:=packer
BINARY=terraform-provider-${NAME}
VERSION=0.3.0
OS_ARCH=linux_amd64

.PHONY: default
default: build

.PHONY: build
build:
	go build -o ${BINARY}

.PHONY: release
release:
	cd bin/${VERSION}/ && \
	zip -r9 ${BINARY}_${VERSION}_darwin_amd64.zip ${BINARY}_v${VERSION}_darwin_amd64 && \
	zip -r9 ${BINARY}_${VERSION}_darwin_arm64.zip ${BINARY}_v${VERSION}_darwin_arm64 && \
	zip -r9 ${BINARY}_${VERSION}_linux_amd64.zip ${BINARY}_v${VERSION}_linux_amd64 && \
	zip -r9 ${BINARY}_${VERSION}_openbsd_amd64.zip ${BINARY}_v${VERSION}_openbsd_amd64 && \
	zip -r9 ${BINARY}_${VERSION}_windows_amd64.zip ${BINARY}_v${VERSION}_windows_amd64 && \
	sha256sum ${BINARY}_${VERSION}_*.zip > ${BINARY}_${VERSION}_SHA256SUMS && \
	gpg --detach-sign ${BINARY}_${VERSION}_SHA256SUMS

.PHONY: build-release
build-release:
	mkdir -p bin/${VERSION}
	GOOS=darwin GOARCH=amd64 go build -o ./bin/${VERSION}/${BINARY}_v${VERSION}_darwin_amd64
	GOOS=darwin GOARCH=arm64 go build -o ./bin/${VERSION}/${BINARY}_v${VERSION}_darwin_arm64
	GOOS=linux GOARCH=amd64 go build -o ./bin/${VERSION}/${BINARY}_v${VERSION}_linux_amd64
	GOOS=openbsd GOARCH=amd64 go build -o ./bin/${VERSION}/${BINARY}_v${VERSION}_openbsd_amd64
	GOOS=windows GOARCH=amd64 go build -o ./bin/${VERSION}/${BINARY}_v${VERSION}_windows_amd64
