.PHONY: build package clean help

VERSION=1.0.0
BINARY=termviewer-agent
DIST_DIR=dist

help:
	@echo "TermViewer Build System"
	@echo "  make build    - Build the Go agent binary"
	@echo "  make package  - Create .deb and .rpm packages using nFPM"
	@echo "  make clean    - Remove build artifacts"

$(DIST_DIR):
	mkdir -p $(DIST_DIR)

build: $(DIST_DIR)
	@echo "Building Agent..."
	cd agent && go build -o ../$(DIST_DIR)/$(BINARY) main.go

package: build
	@echo "Packaging with nFPM..."
	nfpm pkg --packager deb --target $(DIST_DIR)
	nfpm pkg --packager rpm --target $(DIST_DIR)

clean:
	@echo "Cleaning up..."
	rm -rf $(DIST_DIR)
