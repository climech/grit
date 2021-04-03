APPNAME = grit
VERSION = $(shell git describe --long --always --dirty 2>/dev/null || echo -n 'v0.2.0')
GOCMD = go
GOPATH ?= $(shell mktemp -d)
GOMODULE = github.com/climech/grit
CWD = $(shell pwd)
PREFIX ?= /usr
BINDIR ?= $(PREFIX)/bin
BUILDDIR ?= .
BASHCOMPDIR ?= $(PREFIX)/share/bash-completion/completions

all: $(APPNAME)

$(APPNAME):
	@$(GOCMD) build -v \
		-o "$(BUILDDIR)/$(APPNAME)" \
		-ldflags "-s -w -X '$(GOMODULE)/app.Version=$(VERSION)'" \
		"$(CWD)/cmd/$(APPNAME)"

install: $(APPNAME)
	@mkdir -p "$(DESTDIR)$(BINDIR)"
	@install -cv "$(APPNAME)" "$(DESTDIR)$(BINDIR)"

test:
	@$(GOCMD) test -count=1 ./...

clean:
	@rm -f $(APPNAME)

.PHONY: test clean