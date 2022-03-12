GO ?= go
GOFLAGS ?= 
PREFIX ?= /usr/local
BINDIR ?= bin
MANDIR ?= share/man

all: uopds

uopds:
	$(GO) build $(GOFLAGS)

clean:
	rm -f uopds

install:
	mkdir -p $(DESTDIR)$(PREFIX)/$(BINDIR)
	mkdir -p $(DESTDIR)$(PREFIX)/$(MANDIR)/man1
	cp -f uopds $(DESTDIR)$(PREFIX)/$(BINDIR)
	cp -f doc/uopds.1 $(DESTDIR)$(PREFIX)/$(MANDIR)/man1

.PHONY: clean install
