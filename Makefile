BINARY  := kanban
PREFIX  ?= $(HOME)/.local
BINDIR  ?= $(PREFIX)/bin

.PHONY: build install uninstall test clean

build:
	go build -o $(BINARY) ./cmd/kanban

install: build
	install -Dm755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	@echo "installed $(DESTDIR)$(BINDIR)/$(BINARY)"

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)

test:
	go test ./...

clean:
	rm -f $(BINARY)
