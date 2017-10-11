ifndef GOPATH
$(error GOPATH is not set)
endif

SHELL  := env TRAVIS_TAG=$(TRAVIS_TAG) $(SHELL)
TRAVIS_TAG ?= dev

PACKAGE_NAME = github.com/johnmccabe/podtail
LDFLAGS += -X "$(PACKAGE_NAME)/commands.Version=$(TRAVIS_TAG)"
LDFLAGS += -s -w

GOARCH ?= amd64

all: clean build

clean:
	rm -rf podtail-darwin podtail-darwin.tgz podtail podtail.tgz podtail.exe

build: build_darwin_amd64 build_linux_amd64 build_windows_amd64

build_darwin_amd64:
	GOOS=darwin GOARCH=$(GOARCH) CGO_ENABLED=0 go build -a -ldflags '$(LDFLAGS)' -o podtail-darwin

build_linux_amd64:
	GOOS=linux GOARCH=$(GOARCH) CGO_ENABLED=0 go build -a -ldflags '$(LDFLAGS)' -o podtail

build_windows_amd64:
	GOOS=windows GOARCH=$(GOARCH) CGO_ENABLED=0 go build -a -ldflags '$(LDFLAGS)' -o podtail.exe

release: clean release_darwin_amd64 release_linux_amd64 release_windows_amd64

release_darwin_amd64: build_darwin_amd64
	tar czf podtail-darwin.tgz podtail-darwin

release_linux_amd64: build_linux_amd64
	tar czf podtail.tgz podtail

release_windows_amd64: build_windows_amd64

.PHONY: all build build_darwin_amd64 build_linux_amd64 build_windows_amd64