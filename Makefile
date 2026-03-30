BINARY     := cronix
CMD        := ./cmd/cronix
DIST       := dist
GO         := /home/qmz/.local/go-toolchains/go/bin/go

.PHONY: build build-linux-amd64 build-linux-arm64 install test clean

build:
	mkdir -p $(DIST)
	$(GO) build -o $(DIST)/$(BINARY) $(CMD)

build-linux-amd64:
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 $(GO) build -o $(DIST)/$(BINARY)-linux-amd64 $(CMD)

build-linux-arm64:
	mkdir -p $(DIST)
	GOOS=linux GOARCH=arm64 $(GO) build -o $(DIST)/$(BINARY)-linux-arm64 $(CMD)

install: build
	cp $(DIST)/$(BINARY) /usr/local/bin/$(BINARY)

test:
	$(GO) test ./...

clean:
	rm -rf $(DIST)
