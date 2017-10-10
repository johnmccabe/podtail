ifndef GOPATH
$(error GOPATH is not set)
endif

GOARCH ?= amd64
LDFLAGS ?= -s -w

all: clean build

clean:
	rm -rf podtail-darwin podtail podtail.exe

build: build_darwin_amd64 build_linux_amd64 build_windows_amd64

build_darwin_amd64:
	GOOS=darwin GOARCH=$(GOARCH) CGO_ENABLED=0 go build -a -ldflags '$(LDFLAGS)' -o podtail-darwin podtail.go

build_linux_amd64:
	GOOS=linux GOARCH=$(GOARCH) CGO_ENABLED=0 go build -a -ldflags '$(LDFLAGS)' -o podtail podtail.go

build_windows_amd64:
	GOOS=windows GOARCH=$(GOARCH) CGO_ENABLED=0 go build -a -ldflags '$(LDFLAGS)' -o podtail.exe podtail.go

release: clean release_darwin_amd64 release_linux_amd64 release_windows_amd64

release_darwin_amd64: build_darwin_amd64
	tar czf podtail-darwin.tgz podtail-darwin

release_linux_amd64: build_linux_amd64
	tar czf podtail.tgz podtail

release_windows_amd64: build_windows_amd64

.PHONY: all build build_darwin_amd64 build_linux_amd64 build_windows_amd64