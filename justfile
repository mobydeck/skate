version := `git describe --tags --always --dirty 2>/dev/null || echo dev`
module := "skate"
ldflags := "-s -w -X " + module + "/internal/version.Version=" + version
dist := "dist"

# Default recipe
default:
    @just --list

# Build for current platform
build:
    go build -trimpath -ldflags '{{ ldflags }}' -o skate ./cmd/skate

# Run tests
test:
    go test ./...

# Install to ~/.local/bin
install: build
    mkdir -p ~/.local/bin/
    install -m 0755 skate ~/.local/bin/

# Cross-build for all platforms
cross-build: clean
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p {{ dist }}
    platforms=(
        "linux/amd64"
        "linux/arm64"
        "darwin/arm64"
        "windows/amd64"
    )
    for platform in "${platforms[@]}"; do
        IFS='/' read -r goos goarch <<< "$platform"
        output="{{ dist }}/skate-${goos}-${goarch}"
        if [ "$goos" = "windows" ]; then
            output="${output}.exe"
        fi
        echo "Building ${goos}/${goarch}..."
        GOOS=$goos GOARCH=$goarch go build -trimpath -ldflags '{{ ldflags }}' -o "$output" ./cmd/skate
    done
    echo "Done. Binaries in {{ dist }}/"
    ls -lh {{ dist }}/

# Create checksums for dist binaries
checksums:
    #!/usr/bin/env bash
    set -euo pipefail
    cd {{ dist }}
    sha256sum skate-* > checksums.txt
    cat checksums.txt

# Create draft GitHub release for current tag
release: cross-build checksums
    #!/usr/bin/env bash
    set -euo pipefail
    tag="{{ version }}"
    if [ "$tag" = "dev" ] || echo "$tag" | grep -q dirty; then
        echo "Error: cannot release from dev or dirty state. Tag a version first."
        exit 1
    fi

    # Parse GH_TOKEN from ~/.netrc if not set
    if [ -z "${GH_TOKEN:-}" ]; then
        if [ -f ~/.netrc ]; then
            GH_TOKEN=$(grep 'machine github.com' ~/.netrc | head -1 | sed 's/.*password //')
            export GH_TOKEN
        fi
    fi

    if [ -z "${GH_TOKEN:-}" ]; then
        echo "Error: GH_TOKEN not set and not found in ~/.netrc"
        exit 1
    fi

    echo "Creating draft release for ${tag}..."
    gh release create "$tag" \
        --draft \
        --title "Skate ${tag}" \
        --generate-notes \
        {{ dist }}/skate-* \
        {{ dist }}/checksums.txt

    echo "Draft release created: ${tag}"

# Clean build artifacts
clean:
    rm -rf skate {{ dist }}

# Lint
lint:
    golangci-lint run ./...
