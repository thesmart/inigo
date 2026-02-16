# Makefile for github.com/thesmart/inigo
#
# Targets:
#   check        Run formatting, vetting, and tests (pre-release)
#   gate         Run gating scripts: coverage, badges, README update
#   release      Tag and push a new version (VERSION required)
#
# Dry-run options:
#   make gate DRY_RUN=1          Generate artifacts to a temp dir, don't modify README
#   make release DRY_RUN=1       Print what would happen, don't tag or push
#
# Example:
#   make check
#   make gate
#   make release VERSION=v0.1.0

SHELL := /bin/sh

# --- Configuration ---

GATE_DIR     := ./gate
BADGES_DIR   := ./badges
README       := ./README.md
LICENSE_ID   := MIT
MIN_COVERAGE := 80

# Coverage color thresholds
COV_GREEN  := 80
COV_ORANGE := 60

# --- Derived ---

DRY_RUN ?= 0

# --- Phony targets ---

.PHONY: check fmt vet test build build-all gate coverage badges readme release clean help

# --- Default ---

help:
	@echo "Usage:"
	@echo "  make check                Run fmt, vet, and tests"
	@echo "  make gate                 Run gating: coverage, badges, README update"
	@echo "  make gate DRY_RUN=1       Dry-run gate (artifacts to temp dir, no README change)"
	@echo "  make release VERSION=v0.1.0"
	@echo "                            Tag and push a release"
	@echo "  make release VERSION=v0.1.0 DRY_RUN=1"
	@echo "                            Dry-run release (print actions, don't execute)"
	@echo "  make build                Build the inigo CLI binary for the current platform"
	@echo "  make build-all            Cross-compile for all POSIX platforms"
	@echo "  make clean                Remove generated badge artifacts"

# --- Pre-release checks ---

check: fmt vet test

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test -v ./...

build:
	go build -o build/inigo ./cmd/inigo

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 freebsd/amd64

build-all:
	@for platform in $(PLATFORMS); do \
		os=$$(echo "$$platform" | cut -d/ -f1); \
		arch=$$(echo "$$platform" | cut -d/ -f2); \
		output="build/inigo-$${os}-$${arch}"; \
		echo "building $${output}..."; \
		GOOS=$${os} GOARCH=$${arch} go build -o "$${output}" ./cmd/inigo || exit 1; \
	done
	@echo "build-all: complete"

# --- Gating ---

gate: check coverage badges readme
	@echo ""
	@if [ "$(DRY_RUN)" = "1" ]; then \
		echo "gate: dry-run complete (no files modified)"; \
	else \
		echo "gate: complete"; \
	fi

coverage:
	@echo "--- coverage ---"
	@pct=$$(sh $(GATE_DIR)/coverage.sh -r .); \
	echo "coverage: $${pct}%"; \
	if [ "$$(echo "$${pct}" | cut -d. -f1)" -lt "$(MIN_COVERAGE)" ]; then \
		echo "error: coverage $${pct}% is below minimum $(MIN_COVERAGE)%" >&2; \
		exit 1; \
	fi; \
	echo "$${pct}" > /tmp/inigo-coverage-pct.txt

badges: coverage
	@echo "--- badges ---"
	@pct=$$(cat /tmp/inigo-coverage-pct.txt); \
	pct_int=$$(echo "$${pct}" | cut -d. -f1); \
	if [ "$${pct_int}" -ge "$(COV_GREEN)" ]; then \
		cov_color="green"; \
	elif [ "$${pct_int}" -ge "$(COV_ORANGE)" ]; then \
		cov_color="orange"; \
	else \
		cov_color="red"; \
	fi; \
	if [ "$(DRY_RUN)" = "1" ]; then \
		out_dir=$$(mktemp -d); \
		echo "badges: dry-run output -> $${out_dir}"; \
	else \
		out_dir="$(BADGES_DIR)"; \
	fi; \
	sh $(GATE_DIR)/badges.sh \
		-o "$${out_dir}" \
		-p "$${pct_int}" \
		-c "$${cov_color}" \
		-g "A+" \
		-r "green" \
		-l "$(LICENSE_ID)"

readme: badges
	@echo "--- readme ---"
	@pct=$$(cat /tmp/inigo-coverage-pct.txt); \
	pct_int=$$(echo "$${pct}" | cut -d. -f1); \
	badges_md=$$(mktemp); \
	trap 'rm -f "$${badges_md}"' EXIT; \
	sh $(GATE_DIR)/badgesmd.sh \
		-d $(BADGES_DIR) \
		-p "$${pct_int}" \
		-g "A+" \
		-l "$(LICENSE_ID)" \
		-o "$${badges_md}"; \
	if [ "$(DRY_RUN)" = "1" ]; then \
		out_file=$$(mktemp); \
		sh $(GATE_DIR)/mdreplace.sh -t badges -s $(README) -c "$${badges_md}" -o "$${out_file}"; \
		echo "readme: dry-run output -> $${out_file}"; \
		echo "--- dry-run preview (first 15 lines) ---"; \
		head -15 "$${out_file}"; \
	else \
		sh $(GATE_DIR)/mdreplace.sh -t badges -s $(README) -c "$${badges_md}" -o $(README); \
	fi; \
	rm -f "$${badges_md}"

# --- Release ---

release: gate
	@echo ""
	@echo "--- release ---"
	@# Require VERSION
	@if [ -z "$(VERSION)" ]; then \
		echo "error: VERSION is required (e.g. make release VERSION=v0.1.0)" >&2; \
		exit 1; \
	fi
	@# Validate version format
	@echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$' || { \
		echo "error: VERSION must match vMAJOR.MINOR.PATCH (e.g. v0.1.0)" >&2; \
		exit 1; \
	}
	@# Auto-commit gate artifacts (badges, README); fail on other dirty files
	@dirty=$$(git status --porcelain | grep -v ' badges/' | grep -v ' README.md$$'); \
	if [ -n "$${dirty}" ]; then \
		echo "error: working tree has unexpected changes — commit or stash first" >&2; \
		echo "$${dirty}" >&2; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "committing gate artifacts for $(VERSION)..."; \
		git add $(BADGES_DIR) $(README); \
		git commit -m "release $(VERSION)"; \
	fi
	@# Check that we're on main
	@branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$${branch}" != "main" ]; then \
		echo "error: releases must be from the main branch (currently on $${branch})" >&2; \
		exit 1; \
	fi
	@# Check tag doesn't already exist
	@if git rev-parse "$(VERSION)" >/dev/null 2>&1; then \
		echo "error: tag $(VERSION) already exists" >&2; \
		exit 1; \
	fi
	@# Execute or dry-run
	@if [ "$(DRY_RUN)" = "1" ]; then \
		echo "release: dry-run for $(VERSION)"; \
		echo "  would run: git push origin main"; \
		echo "  would run: git tag $(VERSION)"; \
		echo "  would run: git push origin $(VERSION)"; \
		echo "  would run: curl https://proxy.golang.org/github.com/thesmart/inigo/@v/$(VERSION).info"; \
		echo "  would run: make build-all"; \
		echo "  would run: gh release create $(VERSION) build/inigo-* --generate-notes"; \
	else \
		echo "pushing to origin..."; \
		git push origin main; \
		echo "tagging $(VERSION)..."; \
		git tag "$(VERSION)"; \
		echo "pushing tag to origin..."; \
		git push origin "$(VERSION)"; \
		echo "triggering pkg.go.dev indexing..."; \
		curl -sS "https://proxy.golang.org/github.com/thesmart/inigo/@v/$(VERSION).info"; \
		echo ""; \
		echo "building release binaries..."; \
		$(MAKE) build-all; \
		echo "creating GitHub release with binaries..."; \
		gh release create "$(VERSION)" build/inigo-* --generate-notes; \
		echo ""; \
		echo "release: $(VERSION) published"; \
		echo "  https://pkg.go.dev/github.com/thesmart/inigo@$(VERSION)"; \
		echo "  https://github.com/thesmart/inigo/releases/tag/$(VERSION)"; \
	fi

# --- Cleanup ---

clean:
	rm -f $(BADGES_DIR)/*.svg
	rm -f /tmp/inigo-coverage-pct.txt
	rm -rf build/
