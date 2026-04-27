set dotenv-load

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

# Compress a single binary with upx
upx what compression="4":
    upx -{{ compression }} {{ what }}

# Compress linux binaries in dist with upx
compress-linux compression="4":
    #!/usr/bin/env bash
    set -euo pipefail
    for bin in {{ dist }}/skate-linux-*; do
        [ -f "$bin" ] || continue
        echo "Compressing ${bin}..."
        upx -{{ compression }} "$bin"
    done
    echo "Done."
    ls -lh {{ dist }}/skate-linux-*

# Create draft GitHub release for current tag
release: cross-build compress-linux checksums
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

    # Check if release already exists
    if gh release view "$tag" &>/dev/null; then
        echo "Release ${tag} already exists."
        read -p "Delete existing release and create a new one? [y/N] " confirm
        if [ "${confirm,,}" != "y" ]; then
            echo "Aborted."
            exit 0
        fi
        echo "Deleting existing release ${tag}..."
        gh release delete "$tag" --yes
    fi

    echo "Creating draft release for ${tag}..."
    gh release create "$tag" \
        --draft \
        --title "Skate ${tag}" \
        --generate-notes \
        {{ dist }}/skate-* \
        {{ dist }}/checksums.txt

    url=$(gh release view "$tag" --json url -q .url)
    echo "Draft release created: ${url}"

# Clean build artifacts
clean:
    rm -rf skate {{ dist }}

# Generate Homebrew formula from dist binaries
brew-formula: checksums
    #!/usr/bin/env bash
    set -euo pipefail
    tag="{{ version }}"
    if [ "$tag" = "dev" ] || echo "$tag" | grep -q dirty; then
        echo "Error: cannot generate formula from dev or dirty state."
        exit 1
    fi

    template="Formula/skate.rb"
    if [ ! -f "$template" ]; then
        echo "Error: template not found at $template"
        exit 1
    fi

    get_sha() {
        grep "$1" {{ dist }}/checksums.txt | awk '{print $1}'
    }

    sha_darwin_arm64=$(get_sha "skate-darwin-arm64")
    sha_linux_amd64=$(get_sha "skate-linux-amd64")
    sha_linux_arm64=$(get_sha "skate-linux-arm64")

    sed -e "s/VERSION/${tag}/g" \
        -e "s/SHA256_DARWIN_ARM64/${sha_darwin_arm64}/g" \
        -e "s/SHA256_LINUX_AMD64/${sha_linux_amd64}/g" \
        -e "s/SHA256_LINUX_ARM64/${sha_linux_arm64}/g" \
        "$template" > {{ dist }}/skate.rb

    echo "Formula generated: {{ dist }}/skate.rb"
    echo "Copy to your homebrew-tap repo: cp {{ dist }}/skate.rb /path/to/homebrew-tap/Formula/"

# Lint
lint:
    golangci-lint run ./...

commit:
    aicommit -d --model gpt-5-mini
