# Check formatting (exits 1 if any files need formatting).
# Scoped to our source dirs: `.` would descend into .agentic/sources clones.
fmt:
    @if [ -n "$(gofmt -l cmd internal tests)" ]; then gofmt -l cmd internal tests; exit 1; fi

# Fix formatting
fmt-fix:
    gofmt -w cmd internal tests

# Run go vet
vet:
    go vet ./...

# Run unit tests
test:
    go test ./...

# Run integration tests
test-integration:
    go test -tags=integration ./...

# Run a single unit test by name (regexp); optional package path defaults to ./...
test-one PATTERN PATH='./...':
    go test -run {{PATTERN}} {{PATH}}

# Run all tests
test-all: test test-integration

# Run contract tests against real third-party APIs — opt-in, requires $GITHUB_TOKEN and $GITLAB_TOKEN
test-contract:
    @test -n "$GITHUB_TOKEN" || { echo "GITHUB_TOKEN not set"; exit 1; }
    @test -n "$GITLAB_TOKEN" || { echo "GITLAB_TOKEN not set"; exit 1; }
    go test -tags=contract ./...

# Run e2e suite inside Dockerfile.dev with no outbound network
test-e2e: dev-build
    docker run --rm \
        --network none \
        -v "$(pwd)":/home/agent/workspace \
        -v fetch-context-gomod:/go/pkg/mod \
        -v fetch-context-gocache:/home/agent/.cache/go-build \
        --entrypoint bash \
        fetch-context-dev \
        -c 'mkdir -p bin && go build -o ./bin/fetch-context ./cmd/fetch-context && FCBIN=$(pwd)/bin/fetch-context go test -tags=e2e ./tests/e2e/...'

# Build the binary
build:
    mkdir -p bin
    go build -o bin/fetch-context ./cmd/fetch-context

# Install the binary into $GOBIN (or $GOPATH/bin)
install:
    go install ./cmd/fetch-context

# Run the CLI
run *args:
    go run ./cmd/fetch-context {{args}}

# Tidy dependencies
tidy:
    go mod tidy

# Fast pre-push check
gate: fmt vet test test-integration

# Full check (adds the Docker-bound e2e suite)
gate-expensive: gate test-e2e

# Build the dev container image (also used as the sandcastle sandbox image)
dev-build:
    docker build -f Dockerfile.dev \
        --build-arg AGENT_UID=$(id -u) \
        --build-arg AGENT_GID=$(id -g) \
        -t fetch-context-dev .

# Launch a shell in the dev container with agent CLIs (claude, codex, pi) available
dev-shell: dev-build
    @test -f "$HOME/.codex/auth.json" || { echo "missing $HOME/.codex/auth.json — run 'codex login' on the host first"; exit 1; }
    @test -f "$HOME/.pi/agent/auth.json" || { echo "missing $HOME/.pi/agent/auth.json — run 'pi login' on the host first"; exit 1; }
    @test -n "$CLAUDE_CODE_OAUTH_TOKEN" || { echo "CLAUDE_CODE_OAUTH_TOKEN not set in host environment"; exit 1; }
    docker run --rm -it --entrypoint bash \
        -v "$(pwd)":/home/agent/workspace \
        -v fetch-context-gomod:/go/pkg/mod \
        -v fetch-context-gocache:/home/agent/.cache/go-build \
        -v "$HOME/.codex/auth.json":/home/agent/.codex/auth.json:ro \
        -v "$HOME/.pi/agent/auth.json":/home/agent/.pi/agent/auth.json:ro \
        -v "$HOME/.claude/skills":/home/agent/.claude/skills:ro \
        -v "$HOME/.agents/skills":/home/agent/.agents/skills:ro \
        -v "$HOME/.cache/skillvendor":/home/agent/.cache/skillvendor:ro \
        -e CLAUDE_CODE_OAUTH_TOKEN \
        -e GITHUB_TOKEN -e GITLAB_TOKEN \
        fetch-context-dev

# Run sandcastle (reads .sandcastle/main.ts)
sandcastle:
    pnpm exec tsx .sandcastle/main.ts
