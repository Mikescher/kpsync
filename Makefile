
# ./firefox-sync-client
#########################

build: enums
	CGO_ENABLED=0 go build -o _out/kpsync ./cmd/kpsync

run: build
	./_out/ffsclient

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
	GOARCH=386   GOOS=windows               go build -o _out/kpsync_win-386.exe         -tags timetzdata -ldflags "-w -s" ./cmd/cli  # Windows - 32 bit
	GOARCH=amd64 GOOS=windows               go build -o _out/kpsync_win-amd64.exe       -tags timetzdata -ldflags "-w -s" ./cmd/cli  # Windows - 64 bit
	GOARCH=arm64 GOOS=windows               go build -o _out/kpsync_win-arm64.exe       -tags timetzdata -ldflags "-w -s" ./cmd/cli  # Windows - ARM
	GOARCH=amd64 GOOS=darwin                go build -o _out/kpsync_macos-amd64                                           ./cmd/cli  # macOS - 32 bit
	GOARCH=amd64 GOOS=darwin                go build -o _out/kpsync_macos-amd64                                           ./cmd/cli  # macOS - 64 bit
	GOARCH=amd64 GOOS=openbsd               go build -o _out/kpsync_openbsd-amd64                                         ./cmd/cli  # OpenBSD - 64 bit
	GOARCH=arm64 GOOS=openbsd               go build -o _out/kpsync_openbsd-arm64                                         ./cmd/cli  # OpenBSD - ARM
	GOARCH=amd64 GOOS=freebsd               go build -o _out/kpsync_freebsd-amd64                                         ./cmd/cli  # FreeBSD - 64 bit
	GOARCH=arm64 GOOS=freebsd               go build -o _out/kpsync_freebsd-arm64                                         ./cmd/cli  # FreeBSD - ARM
