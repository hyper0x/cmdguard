VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"

.PHONY: build install clean

build:
	go build $(LDFLAGS) -o cmdguard .

install:
	go install $(LDFLAGS) .

clean:
	rm -f cmdguard
