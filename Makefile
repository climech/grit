APPNAME = grit
VERSION = $(shell git describe --long --always --dirty 2>/dev/null || echo -n 'v0.1-git')
GOCMD = go
GOPATH ?= $(shell mktemp -d)
GOMODULE = github.com/climech/grit
CWD = $(shell pwd)
PREFIX ?= /usr
BINDIR ?= $(PREFIX)/bin
BUILDDIR ?= .
BASHCOMPDIR ?= $(PREFIX)/share/bash-completion/completions

all: build

build:
	@$(GOCMD) build -v \
		-o "$(BUILDDIR)/$(APPNAME)" \
		-ldflags "-X '$(GOMODULE)/app.Version=$(VERSION)'" \
		"$(CWD)/cmd/$(APPNAME)"

install: grit
	@install -v -D -t $(DESTDIR)$(BINDIR) $^

test:
	@$(GOCMD) test -count=1 ./...

clean: grit
	@rm -f grit

.PHONY: build test