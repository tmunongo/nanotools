.PHONY: help build-templ pre_commit version bump-patch bump-minor bump-major patch minor major tag push-tags release-patch release-minor release-major

# Default target displays help
help:
	@echo "Nanotools Make Commands:"
	@echo "  make help             - Show this help message"
	@echo "  make build-templ      - Generate Templ templates"
	@echo "  make pre_commit       - Format and vet Go code"
	@echo "  make version          - Display current latest semver tag"
	@echo "  make bump-patch       - Bump patch version (vX.Y.Z -> vX.Y.Z+1) and tag"
	@echo "  make bump-minor       - Bump minor version (vX.Y.Z -> vX.Y+1.0) and tag"
	@echo "  make bump-major       - Bump major version (vX.Y.Z -> vX+1.0.0) and tag"
	@echo "  make tag VERSION=v... - Tag an explicit version (e.g. make tag VERSION=v1.0.0)"
	@echo "  make push-tags        - Push all tags to origin"
	@echo "  make release-patch    - Bump patch, tag, and immediately push to origin"
	@echo "  make release-minor    - Bump minor, tag, and immediately push to origin"
	@echo "  make release-major    - Bump major, tag, and immediately push to origin"

.PHONY: help build-templ pre_commit version bump-patch bump-minor bump-major patch minor major tag push-tags release-patch release-minor release-major

# Default target displays help
help:
	@echo "Nanotools Make Commands:"
	@echo "  make help             - Show this help message"
	@echo "  make build-templ      - Generate Templ templates"
	@echo "  make pre_commit       - Format and vet Go code"
	@echo "  make version          - Display current latest semver tag"
	@echo "  make bump-patch       - Bump patch version (vX.Y.Z -> vX.Y.Z+1) and tag"
	@echo "  make bump-minor       - Bump minor version (vX.Y.Z -> vX.Y+1.0) and tag"
	@echo "  make bump-major       - Bump major version (vX.Y.Z -> vX+1.0.0) and tag"
	@echo "  make tag VERSION=v... - Tag an explicit version (e.g. make tag VERSION=v1.0.0)"
	@echo "  make push-tags        - Push all tags to origin"
	@echo "  make release-patch    - Bump patch, tag, and immediately push to origin"
	@echo "  make release-minor    - Bump minor, tag, and immediately push to origin"
	@echo "  make release-major    - Bump major, tag, and immediately push to origin"

build-templ:
	templ generate

pre_commit:
	go fmt ./...
	go vet ./...

# Display the current latest git tag
version:
	@latest=$$(git tag -l --sort=-v:refname "v*" 2>/dev/null | head -n 1); \
	if [ -z "$$latest" ]; then \
		echo "No tags found. Baseline is v0.0.0"; \
	else \
		echo "Current version: $$latest"; \
	fi

# Helper macro to calculate and apply semver bump
define bump_semver
	latest=$$(git tag -l --sort=-v:refname "v*" 2>/dev/null | head -n 1); \
	if [ -z "$$latest" ]; then latest="v0.0.0"; fi; \
	raw=$${latest#v}; \
	major=$$(echo "$$raw" | cut -d. -f1); \
	minor=$$(echo "$$raw" | cut -d. -f2); \
	patch=$$(echo "$$raw" | cut -d. -f3); \
	major=$${major:-0}; \
	minor=$${minor:-0}; \
	patch=$${patch:-0}; \
	if [ "$(1)" = "major" ]; then \
		next="v$$((major + 1)).0.0"; \
	elif [ "$(1)" = "minor" ]; then \
		next="v$${major}.$$((minor + 1)).0"; \
	else \
		next="v$${major}.$${minor}.$$((patch + 1))"; \
	fi; \
	if [ -n "$$(git status --porcelain)" ]; then \
		echo "Warning: Working directory has uncommitted changes."; \
	fi; \
	echo "Bumping $$latest -> $$next"; \
	git tag -a "$$next" -m "Release $$next" && \
	echo "Successfully created tag $$next." && \
	echo "Run 'git push origin $$next' or 'make push-tags' to push and trigger GHCR build."
endef

bump-patch:
	@$(call bump_semver,patch)

bump-minor:
	@$(call bump_semver,minor)

bump-major:
	@$(call bump_semver,major)

# Convenient shorthand aliases
patch: bump-patch
minor: bump-minor
major: bump-major

# Tag an explicit version: make tag VERSION=v1.0.0
tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Example: make tag VERSION=v1.0.0"; \
		exit 1; \
	fi; \
	if [ -n "$$(git status --porcelain)" ]; then \
		echo "Warning: Working directory has uncommitted changes."; \
	fi; \
	git tag -a "$(VERSION)" -m "Release $(VERSION)" && \
	echo "Successfully created tag $(VERSION)." && \
	echo "Run 'git push origin $(VERSION)' or 'make push-tags' to push and trigger GHCR build."

# Push tags to remote
push-tags:
	git push origin --tags

# Bump and immediately push to trigger GHCR release
release-patch: bump-patch
	@latest=$$(git tag -l --sort=-v:refname "v*" 2>/dev/null | head -n 1); \
	echo "Pushing $$latest to origin..."; \
	git push origin "$$latest"

release-minor: bump-minor
	@latest=$$(git tag -l --sort=-v:refname "v*" 2>/dev/null | head -n 1); \
	echo "Pushing $$latest to origin..."; \
	git push origin "$$latest"

release-major: bump-major
	@latest=$$(git tag -l --sort=-v:refname "v*" 2>/dev/null | head -n 1); \
	echo "Pushing $$latest to origin..."; \
	git push origin "$$latest"

# Display the current latest git tag
version:
	@latest=$$(git tag -l --sort=-v:refname "v*" 2>/dev/null | head -n 1); \
	if [ -z "$$latest" ]; then \
		echo "No tags found. Baseline is v0.0.0"; \
	else \
		echo "Current version: $$latest"; \
	fi

# Helper macro to calculate and apply semver bump
define bump_semver
	latest=$$(git tag -l --sort=-v:refname "v*" 2>/dev/null | head -n 1); \
	if [ -z "$$latest" ]; then latest="v0.0.0"; fi; \
	raw=$${latest#v}; \
	major=$$(echo "$$raw" | cut -d. -f1); \
	minor=$$(echo "$$raw" | cut -d. -f2); \
	patch=$$(echo "$$raw" | cut -d. -f3); \
	major=$${major:-0}; \
	minor=$${minor:-0}; \
	patch=$${patch:-0}; \
	if [ "$(1)" = "major" ]; then \
		next="v$$((major + 1)).0.0"; \
	elif [ "$(1)" = "minor" ]; then \
		next="v$${major}.$$((minor + 1)).0"; \
	else \
		next="v$${major}.$${minor}.$$((patch + 1))"; \
	fi; \
	if [ -n "$$(git status --porcelain)" ]; then \
		echo "Warning: Working directory has uncommitted changes."; \
	fi; \
	echo "Bumping $$latest -> $$next"; \
	git tag -a "$$next" -m "Release $$next" && \
	echo "Successfully created tag $$next." && \
	echo "Run 'git push origin $$next' or 'make push-tags' to push and trigger GHCR build."
endef

bump-patch:
	@$(call bump_semver,patch)

bump-minor:
	@$(call bump_semver,minor)

bump-major:
	@$(call bump_semver,major)

# Convenient shorthand aliases
patch: bump-patch
minor: bump-minor
major: bump-major

# Tag an explicit version: make tag VERSION=v1.0.0
tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Example: make tag VERSION=v1.0.0"; \
		exit 1; \
	fi; \
	if [ -n "$$(git status --porcelain)" ]; then \
		echo "Warning: Working directory has uncommitted changes."; \
	fi; \
	git tag -a "$(VERSION)" -m "Release $(VERSION)" && \
	echo "Successfully created tag $(VERSION)." && \
	echo "Run 'git push origin $(VERSION)' or 'make push-tags' to push and trigger GHCR build."

# Push tags to remote
push-tags:
	git push origin --tags

# Bump and immediately push to trigger GHCR release
release-patch: bump-patch
	@latest=$$(git tag -l --sort=-v:refname "v*" 2>/dev/null | head -n 1); \
	echo "Pushing $$latest to origin..."; \
	git push origin "$$latest"

release-minor: bump-minor
	@latest=$$(git tag -l --sort=-v:refname "v*" 2>/dev/null | head -n 1); \
	echo "Pushing $$latest to origin..."; \
	git push origin "$$latest"

release-major: bump-major
	@latest=$$(git tag -l --sort=-v:refname "v*" 2>/dev/null | head -n 1); \
	echo "Pushing $$latest to origin..."; \
	git push origin "$$latest"