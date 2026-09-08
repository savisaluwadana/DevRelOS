// This directory holds the Next.js app and contains no Go source of its own.
//
// It is declared as a separate Go module purely so the root module's ./...
// pattern stops descending into it. After `npm install`, apps/web/node_modules
// contains third-party Go packages shipped inside npm dependencies (for
// example flatted/golang), and `go build ./...` / `go test ./...` compiled
// them as part of this repository - unnecessary work and an unwanted
// supply-chain surface in the Go build.
//
// Nothing is expected to import this module.
module github.com/savisaluwadana/DevRelOS/apps/web

go 1.25.0
