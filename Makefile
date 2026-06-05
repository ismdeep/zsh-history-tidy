BINDIR ?= /usr/local/bin
PROGRAM ?= zsh-history-tidy
BUILD_DIR ?= ./build
LOCAL_BINARY ?= $(BUILD_DIR)/$(PROGRAM)
VERSION ?= 0.0.0

# `make help`
.PHONY: help
help:
	@cat Makefile | grep '# `' | grep -v '@cat Makefile'

# `make build`
.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -o $(BUILD_DIR)/$(PROGRAM)_linux_amd64  -mod vendor -trimpath -ldflags '-s -w -X main.Version=$(VERSION)' .
	CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build -o $(BUILD_DIR)/$(PROGRAM)_linux_arm64  -mod vendor -trimpath -ldflags '-s -w -X main.Version=$(VERSION)' .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(PROGRAM)_darwin_amd64 -mod vendor -trimpath -ldflags '-s -w -X main.Version=$(VERSION)' .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(PROGRAM)_darwin_arm64 -mod vendor -trimpath -ldflags '-s -w -X main.Version=$(VERSION)' .

# `make install`
.PHONY: install
install:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -o $(LOCAL_BINARY) -mod vendor -trimpath -ldflags '-s -w -X main.Version=$(VERSION)' .
	mkdir -p $(DESTDIR)$(BINDIR)
	install -m 755 $(LOCAL_BINARY) $(DESTDIR)$(BINDIR)/$(PROGRAM)

# `make clean`
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
