APPNAME = grit
VERSION = $(shell git describe --long --always --dirty 2>/dev/null || echo -n 'v0.2.0')
GOCMD = go
GOPATH ?= $(shell mktemp -d)
GOMODULE = github.com/climech/grit
CWD = $(shell pwd)
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
BUILDDIR ?= .
BASHCOMPDIR ?= $(PREFIX)/share/bash-completion/completions
GOLANG_CROSS_VERSION  ?= v1.16.3

all: build

.PHONY: build
build:
	@$(GOCMD) build -v \
		-o "$(BUILDDIR)/$(APPNAME)" \
		-ldflags "-s -w -X '$(GOMODULE)/app.Version=$(VERSION)'" \
		"$(CWD)/cmd/$(APPNAME)"

install: $(APPNAME)
	@mkdir -p "$(DESTDIR)$(BINDIR)"
	@install -cv "$(APPNAME)" "$(DESTDIR)$(BINDIR)"

.PHONY: test
test:
	@$(GOCMD) test -count=1 ./...

.PHONY: clean
clean:
	@rm -f $(APPNAME)
	
.PHONY: release-dry-run
release-dry-run:
	@docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(GOMODULE) \
		-w /go/src/$(GOMODULE) \
		troian/golang-cross:${GOLANG_CROSS_VERSION} \
		--rm-dist --snapshot

.PHONY: release
release:
	@if [ ! -f ".release-env" ]; then \
		echo "\033[91m.release-env is required for release\033[0m";\
		exit 1;\
	fi
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		--env-file .release-env \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(GOMODULE) \
		-w /go/src/$(GOMODULE) \
		troian/golang-cross:${GOLANG_CROSS_VERSION} \
		release --rm-dist