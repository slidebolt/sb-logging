# Git Workflow for sb-logging

This repository contains the Slidebolt Logging service, which provides structured logging, persistent storage (SQLite), and real-time log distribution. It produces a standalone binary.

## Dependencies
- **Internal:**
  - `sb-contract`: Core interfaces and shared structures.
  - `sb-logging-sdk`: Client interfaces for logging (self-referenced for implementation).
  - `sb-messenger-sdk`: Shared messaging interfaces and NATS implementation.
  - `sb-runtime`: Core execution environment and logging integration.
- **External:** 
  - `modernc.org/sqlite`: Pure Go SQLite implementation.

## Build Process
- **Type:** Go Application (Service).
- **Consumption:** Run as the central logging service for Slidebolt.
- **Artifacts:** Produces a binary named `sb-logging`.
- **Command:** `go build -o sb-logging ./cmd/sb-logging`
- **Validation:** 
  - Validated through unit tests: `go test -v ./...`
  - Validated by successful compilation of the binary.

## Pre-requisites & Publishing
As the central logging service, `sb-logging` must be updated whenever its internal dependencies are changed.

**Before publishing:**
1. Determine current tag: `git tag | sort -V | tail -n 1`
2. Ensure all local tests pass: `go test -v ./...`
3. Ensure the binary builds: `go build -o sb-logging ./cmd/sb-logging`

**Publishing Order:**
1. Ensure `sb-contract`, `sb-logging-sdk`, `sb-messenger-sdk`, and `sb-runtime` are tagged and pushed.
2. Update `sb-logging/go.mod` to reference the latest tags.
3. Determine next semantic version for `sb-logging` (e.g., `v1.0.0`).
4. Commit and push the changes to `main`.
5. Tag the repository: `git tag v1.0.0`.
6. Push the tag: `git push origin main v1.0.0`.

## Update Workflow & Verification
1. **Modify:** Update logging service logic in `app/`, `internal/`, or `server/`.
2. **Verify Local:**
   - Run `go mod tidy`.
   - Run `go test ./...`.
   - Run `go build -o sb-logging ./cmd/sb-logging`.
3. **Commit:** Ensure the commit message clearly describes the logging service change.
4. **Tag & Push:** (Follow the Publishing Order above).
