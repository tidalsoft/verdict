# verdict build/test gate. This module is a pure library with no binaries:
# `make check` is build, vet, lint, unit tests, integration tests, and
# coverage enforced at 100% file/package/total with no exclusions. CI
# invokes these same targets rather than re-implementing the loop, so a
# green `make check` locally means a green CI.
#
# Tool versions are pinned deliberately (no @latest) -- a project whose pitch
# is determinism does not lint or gate coverage against a moving target.
GOLANGCI_LINT_VERSION := v2.12.2
GO_TEST_COVERAGE_VERSION := v2.19.0

# `go install` puts pinned tools (golangci-lint, go-test-coverage) in GOBIN,
# or GOPATH/bin if GOBIN is unset -- and neither directory is on PATH by
# default on a fresh checkout. Without this, `make check` fails with a
# tooling error that reads like a lint failure rather than what it actually
# is: this Makefile not looking in the place its own tools were installed.
# Appended after the caller's PATH, not prepended, so a tool a developer has
# deliberately placed earlier on PATH still wins over the pinned one here.
GO_TOOL_BIN := $(shell go env GOBIN)
ifeq ($(GO_TOOL_BIN),)
GO_TOOL_BIN := $(shell go env GOPATH)/bin
endif
export PATH := $(PATH):$(GO_TOOL_BIN)

COVER_DIR := coverage
COVER_PROFILE := $(COVER_DIR)/cover.out

# Set RACE=1 to run tests under the race detector, e.g. `make test RACE=1`.
# CI always sets it (see .github/workflows/ci.yml), so this is the one place
# that knows how to run tests with or without -race -- CI calls this target
# rather than hand-rolling its own loop.
RACE_FLAG :=
ifeq ($(RACE),1)
RACE_FLAG := -race
endif

.PHONY: build vet test test-integration lint cover check clean \
	release-check release-bump release

build:
	go build ./...

vet:
	go vet ./...

test:
	go test $(RACE_FLAG) ./...

test-integration:
	go test $(RACE_FLAG) -tags integration ./...

# golangci-lint is never silently skipped: if it isn't on PATH this fails
# loudly with the install command, rather than letting `make check` report
# green without having linted anything.
# The tool invocation below is folded into the same shell line as the
# existence check (`fi; \` rather than a fresh recipe line) rather than left
# as a standalone command. Recipe lines with no shell metacharacters get
# exec'd directly by make instead of via a shell, and on the GNU Make 3.81
# that macOS still ships, that direct-exec path does not see the PATH this
# Makefile just exported -- it would report golangci-lint missing even
# though the check above just found it. Sharing the shell invocation avoids
# relying on a second, differently-behaved lookup for the same tool.
lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found on PATH or in $(GO_TOOL_BIN)."; \
		echo ".golangci.yml uses the v2 config schema; install the pinned v2 version:"; \
		echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)"; \
		exit 1; \
	fi; \
	echo "golangci-lint run ./..."; \
	golangci-lint run ./...

# Produces a coverage profile AND enforces the threshold in .testcoverage.yml
# (file/package/total all 100%, with no exclusions -- this module has no
# binaries, so there is no main.go carve-out to make). This is a gate, not a
# report: it fails the build below bar.
cover:
	@mkdir -p $(COVER_DIR)
	go test -coverprofile=$(COVER_PROFILE) ./...
	@if ! command -v go-test-coverage >/dev/null 2>&1; then \
		echo "go-test-coverage not found on PATH or in $(GO_TOOL_BIN)."; \
		echo "Install the pinned version:"; \
		echo "  go install github.com/vladopajic/go-test-coverage/v2@$(GO_TEST_COVERAGE_VERSION)"; \
		exit 1; \
	fi; \
	echo "go-test-coverage --config=.testcoverage.yml"; \
	go-test-coverage --config=.testcoverage.yml

check: build vet lint test test-integration cover

clean:
	rm -rf $(COVER_DIR)

# --- Release discipline ------------------------------------------------
# A consumer's go.mod must require a published verdict tag, never a replace
# directive: a replace makes one machine's checkout layout part of the build
# and hides whether the tag anyone else resolves is even buildable. These
# targets enforce that workflow — release-check validates preconditions, release-
# bump updates the const, release runs the full ceremony.
# -----------------------------------------------------------------------

release-check:
ifndef VERSION
	$(error VERSION is required — e.g. make release-check VERSION=0.2.0)
endif
	@if ! echo "$(VERSION)" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "VERSION=$(VERSION) is not valid semver (expected X.Y.Z)."; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Working tree is dirty — commit or stash before releasing."; \
		exit 1; \
	fi
	@if [ "$$(git rev-parse --abbrev-ref HEAD)" != "main" ]; then \
		echo "Not on main (currently on $$(git rev-parse --abbrev-ref HEAD)) — switch to main."; \
		exit 1; \
	fi
	@if git rev-parse -q --verify refs/tags/v$(VERSION) >/dev/null 2>&1; then \
		echo "Tag v$(VERSION) already exists — pick a different VERSION."; \
		exit 1; \
	fi
	@echo "release-check OK: VERSION=$(VERSION), tree clean, on main, tag absent."

release-bump: release-check
	sed -i '' -e 's/^const Version = ".*"/const Version = "$(VERSION)"/' version.go
	@echo "Bumped version.go to $(VERSION). Review with: git diff version.go"

release: release-check release-bump
	go test ./...
	git add version.go version_test.go
	git commit -m "release: v$(VERSION)"
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin main
	git push origin v$(VERSION)
	@echo "Released v$(VERSION)."
