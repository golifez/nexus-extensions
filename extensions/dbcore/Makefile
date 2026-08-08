.PHONY: build test package

VERSION ?= 0.1.0
OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)
DIST := ../../dist/dbcore/$(VERSION)/$(OS)-$(ARCH)

build:
	mkdir -p $(DIST)/bin
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -trimpath -ldflags="-s -w" -o $(DIST)/bin/nexus-extension-dbcore ./cmd/dbcore
	cp manifest.json README.md $(DIST)/

test:
	go test ./cmd/dbcore

package: build
	cd $(DIST) && tar -czf ../../nexus-extension-dbcore-$(VERSION)-$(OS)-$(ARCH).tar.gz manifest.json README.md bin
