VERSION ?= $(shell sed -n 's/^var Version = "\(.*\)"/\1/p' internal/version/version.go)
BUILD_DIR := build
GO ?= go
ZIP ?= zip -j
LDFLAGS := -s -w -X github.com/rguziy/ndrop/internal/version.Version=$(VERSION)

# Client and server packages/binary names
CLIENT_PKG := ./cmd/ndrop
SERVER_PKG := ./cmd/ndropd
CLIENT_BIN := ndrop
SERVER_BIN := ndropd

TARGETS := \
	linux-amd64 \
	linux-armv5 \
	linux-armv6 \
	linux-armv7 \
	windows-amd64 \
	darwin-amd64 \
	darwin-arm64

.DEFAULT_GOAL := help

.PHONY: all clean release help $(TARGETS) build

all: release

release: $(TARGETS)

clean:
	rm -rf $(BUILD_DIR)

help:
	@printf "Version: %s\n\n" "$(VERSION)"
	@printf "Usage:\n"
	@printf "  make            # alias for release\n"
	@printf "  make release    # build all targets\n"
	@printf "  make clean      # remove build artifacts\n"
	@printf "  make <target>   # build one target\n\n"
	@printf "Targets:\n"
	@printf "  %s\n" "$(TARGETS)"

linux-amd64: GOOS=linux
linux-amd64: GOARCH=amd64
linux-amd64: EXT=
linux-amd64: TARGET=linux-amd64
linux-amd64: build

linux-armv5: GOOS=linux
linux-armv5: GOARCH=arm
linux-armv5: GOARM=5
linux-armv5: EXT=
linux-armv5: TARGET=linux-armv5
linux-armv5: build

linux-armv6: GOOS=linux
linux-armv6: GOARCH=arm
linux-armv6: GOARM=6
linux-armv6: EXT=
linux-armv6: TARGET=linux-armv6
linux-armv6: build

linux-armv7: GOOS=linux
linux-armv7: GOARCH=arm
linux-armv7: GOARM=7
linux-armv7: EXT=
linux-armv7: TARGET=linux-armv7
linux-armv7: build

windows-amd64: GOOS=windows
windows-amd64: GOARCH=amd64
windows-amd64: EXT=.exe
windows-amd64: TARGET=windows-amd64
windows-amd64: build

darwin-amd64: GOOS=darwin
darwin-amd64: GOARCH=amd64
darwin-amd64: EXT=
darwin-amd64: TARGET=darwin-amd64
darwin-amd64: build

darwin-arm64: GOOS=darwin
darwin-arm64: GOARCH=arm64
darwin-arm64: EXT=
darwin-arm64: TARGET=darwin-arm64
darwin-arm64: build

# build target builds both client and server, then zips them together
build:
	@rm -rf $(BUILD_DIR)/$(TARGET)
	@mkdir -p $(BUILD_DIR)/$(TARGET)
	@rm -f $(BUILD_DIR)/$(CLIENT_BIN)-$(TARGET)-$(VERSION).zip
	@printf "Building client and server for %s/%s%s\n" "$(GOOS)" "$(GOARCH)" "$(if $(GOARM), GOARM=$(GOARM),)"
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) $(GO) build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(TARGET)/$(CLIENT_BIN)$(EXT) $(CLIENT_PKG)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) $(GO) build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(TARGET)/$(SERVER_BIN)$(EXT) $(SERVER_PKG)
	@cd $(BUILD_DIR)/$(TARGET) && $(ZIP) ../$(CLIENT_BIN)-$(TARGET)-$(VERSION).zip $(CLIENT_BIN)$(EXT) $(SERVER_BIN)$(EXT)
	@rm -rf $(BUILD_DIR)/$(TARGET)
