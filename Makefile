
# ./firefox-sync-client
#########################

build: enums
	CGO_ENABLED=0 go build -o _out/kpsync ./cmd/cli

run: build
	./_out/kpsync

clean:
	go clean
	rm -rf ./_out/*

enums:
	go generate ./...

package:
	#
	# Manually do beforehand:
	#   - Update version in version.go
	#   - Create tag
	#   - Commit
	#

	go clean
	rm -rf ./_out/*

	GOARCH=386   GOOS=linux   CGO_ENABLED=0 go build -o _out/kpsync_linux-386-static                                      ./cmd/cli  # Linux - 32 bit
	GOARCH=amd64 GOOS=linux   CGO_ENABLED=0 go build -o _out/kpsync_linux-amd64-static                                    ./cmd/cli  # Linux - 64 bit
	GOARCH=arm64 GOOS=linux   CGO_ENABLED=0 go build -o _out/kpsync_linux-arm64-static                                    ./cmd/cli  # Linux - ARM
	GOARCH=386   GOOS=linux                 go build -o _out/kpsync_linux-386                                             ./cmd/cli  # Linux - 32 bit
	GOARCH=amd64 GOOS=linux                 go build -o _out/kpsync_linux-amd64                                           ./cmd/cli  # Linux - 64 bit
	GOARCH=arm64 GOOS=linux                 go build -o _out/kpsync_linux-arm64                                           ./cmd/cli  # Linux - ARM
