build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./build/layerd-linux-amd64 ./cmd/layerd
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./build/layerd-linux-arm64 ./cmd/layerd

do-checksum-linux:
	cd build && shasum -a 256 \
    	layerd-linux-amd64 layerd-linux-arm64 \
    	> layerd-checksum-linux

build-linux-with-checksum: build-linux do-checksum-linux

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./build/layerd-darwin-amd64 ./cmd/layerd
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ./build/layerd-darwin-arm64 ./cmd/layerd

build-all: build-linux build-darwin

do-checksum-darwin:
	cd build && shasum -a 256 \
		layerd-darwin-amd64 layerd-darwin-arm64 \
		> layer-checksum-darwin

build-darwin-with-checksum: build-darwin do-checksum-darwin

build-with-checksum: build-linux-with-checksum build-darwin-with-checksum
