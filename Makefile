PREFIX ?= ~/.local
.PHONY: install uninstall clean

GOFILES = $(shell find . -name '*.go')

all: build install
build: $(GOFILES)
	@mkdir -p dist
	@CGO_ENABLED=0 go build -ldflags="-s -w -extldflags '-static'" -o dist/geminicommit
	@dist/geminicommit completion bash >dist/geminicommit.bash
	@dist/geminicommit completion zsh >dist/_geminicommit.zsh
	@dist/geminicommit completion fish >dist/geminicommit.fish
	@echo "Build complete"

clean:
	rm -rf dist

install:
	install -Dm755 dist/geminicommit $(PREFIX)/bin/geminicommit
	install -Dm644 dist/geminicommit.bash $(PREFIX)/share/bash-completion/completions/geminicommit
	install -Dm644 dist/_geminicommit.zsh $(PREFIX)/share/zsh/site-functions/_geminicommit
	install -Dm644 dist/geminicommit.fish $(PREFIX)/share/fish/vendor_completions.d/geminicommit.fish

uninstall:
	rm -f $(PREFIX)/bin/geminicommit
	rm -f $(PREFIX)/share/bash-completion/completions/geminicommit
	rm -f $(PREFIX)/share/zsh/site-functions/_geminicommit
	rm -f $(PREFIX)/share/fish/vendor_completions.d/geminicommit.fish
