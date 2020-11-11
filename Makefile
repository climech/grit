APPNAME = grit
VERSION = $(shell git describe --long --always --dirty 2>/dev/null || echo -n 'v0.1-git')
GO = go
GOMODULE = github.com/climech/grit
GOPATH ?= $(shell mktemp -d)
CWD = $(shell pwd)
PREFIX ?= /usr
BINDIR ?= $(PREFIX)/bin
BUILDDIR = .
BASHCOMPDIR ?= $(PREFIX)/share/bash-completion/completions

all: build

build: cmd/*
	@for name in $^; do \
		env GOPATH=${GOPATH} GOOS=${GOOS} GOARCH=${GOARCH} GOARM=${GOARM} \
		${GO} build -v \
			-o ${BUILDDIR}/$$(basename $$name)${SUFFIX} \
			-ldflags "-X '${GOMODULE}/app.Version=${VERSION}'" \
			"${CWD}/$$name" \
			&& echo "-> ${BUILDDIR}/$$(basename $$name)${SUFFIX}" \
			|| echo "** Failed to build $$name **" 1>&2; \
	done

install: grit
	@install -v -D -t ${DESTDIR}${BINDIR} $^

clean: grit
	@rm -rf grit

.PHONY: build