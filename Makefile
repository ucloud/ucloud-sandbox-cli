.PHONY: install build dev link test clean help publish patch minor major

# Install dependencies
install:
	pnpm install

# Build the CLI
build:
	pnpm build

# Development mode with watch
dev:
	pnpm dev

# Link CLI globally
link: build
	pnpm link --global

# Run tests
test:
	pnpm test

# Clean build artifacts
clean:
	rm -rf dist

# Full rebuild
rebuild: clean build

# Quick build and test
quick: build
	./dist/index.js --help

# Login
login:
	./dist/index.js auth login

# List templates
tpl-list:
	./dist/index.js tpl list

# Version bump (patch: 0.1.0 -> 0.1.1)
patch:
	npm version patch --no-git-tag-version

# Version bump (minor: 0.1.0 -> 0.2.0)
minor:
	npm version minor --no-git-tag-version

# Version bump (major: 0.1.0 -> 1.0.0)
major:
	npm version major --no-git-tag-version

# Publish to npm (auto bump patch version)
publish: patch build
	npm publish --access public

# Publish without version bump
publish-only: build
	npm publish --access public

# Help
help:
	@echo "Available targets:"
	@echo "  install      - Install dependencies"
	@echo "  build        - Build the CLI"
	@echo "  dev          - Start development mode with watch"
	@echo "  link         - Build and link CLI globally"
	@echo "  test         - Run tests"
	@echo "  clean        - Remove build artifacts"
	@echo "  rebuild      - Clean and rebuild"
	@echo "  quick        - Build and show help"
	@echo "  login        - Run auth login"
	@echo "  tpl-list     - List templates"
	@echo "  patch        - Bump patch version (0.1.0 -> 0.1.1)"
	@echo "  minor        - Bump minor version (0.1.0 -> 0.2.0)"
	@echo "  major        - Bump major version (0.1.0 -> 1.0.0)"
	@echo "  publish      - Bump patch version and publish to npm"
	@echo "  publish-only - Publish to npm without version bump"
	@echo "  help         - Show this help"
