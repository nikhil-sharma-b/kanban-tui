BINARY  := kanban
PREFIX  ?= $(HOME)/.local
BINDIR  ?= $(PREFIX)/bin
SKILLDIR ?= $(HOME)/.claude/skills

.PHONY: build install install-skill uninstall test clean

build:
	go build -o $(BINARY) ./cmd/kanban

install: build
	install -Dm755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	@echo "installed $(DESTDIR)$(BINDIR)/$(BINARY)"

# Symlinks the agent skill so `git pull` keeps it current.
install-skill:
	mkdir -p $(SKILLDIR)
	ln -sfn $(CURDIR)/skills/kanban $(SKILLDIR)/kanban
	@echo "linked $(SKILLDIR)/kanban -> $(CURDIR)/skills/kanban"

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)
	if [ -L $(SKILLDIR)/kanban ]; then rm $(SKILLDIR)/kanban; fi

test:
	go test ./...

clean:
	rm -f $(BINARY)
