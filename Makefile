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
lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found on PATH."; \
		echo ".golangci.yml uses the v2 config schema; install the pinned v2 version:"; \
		echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)"; \
		exit 1; \
	fi
	golangci-lint run ./...

# Produces a coverage profile AND enforces the threshold in .testcoverage.yml
# (file/package/total all 100%, with no exclusions -- this module has no
# binaries, so there is no main.go carve-out to make). This is a gate, not a
# report: it fails the build below bar.
cover:
	@mkdir -p $(COVER_DIR)
	go test -coverprofile=$(COVER_PROFILE) ./...
	@if ! command -v go-test-coverage >/dev/null 2>&1; then \
		echo "go-test-coverage not found on PATH."; \
		echo "Install the pinned version:"; \
		echo "  go install github.com/vladopajic/go-test-coverage/v2@$(GO_TEST_COVERAGE_VERSION)"; \
		exit 1; \
	fi
	go-test-coverage --config=.testcoverage.yml

check: build vet lint test test-integration cover

clean:
	rm -rf $(COVER_DIR)

# --- Release discipline ------------------------------------------------
# gatepost/PLAN.md "Verdict release discipline": go.mod must require a
# published verdict tag, never a replace directive (U-3). These targets
# enforce that workflow — release-check validates preconditions, release-
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
