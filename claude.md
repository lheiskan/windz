# Instructions for Claude

This is a repository for API and tools to view wind data observations from FMI stations.

API for accessing FMI data is in pkg/fmi/ directory. It's a stateless API wrapping calls to FMI endpoints.

## Please take the following into account

* When you generate a code file, include a comment with `@vibe: 🤖 -- ai`
* Prefer functionality-based grouping of files over grouping by technical layer
* Don't create backward-compatible implementations when refactoring code  
* If you need to interactively browse a web page, use the Playwright MCP tool

## Testing

* Run tests before committing changes with `go test ./...`
* Use integration tests sparingly - add `RUN_INTEGRATION_TESTS=true` flag when needed
* Include benchmarks for performance-critical code

## Git Commits

* Use conventional commit format: `feat:`, `fix:`, `docs:`, `refactor:`
* Run `gofmt -w` on Go files before committing (pre-commit hook enforces this)
* Include performance metrics in commit messages when relevant

## FMI API Integration

* Organize by functionality (observations/, stations/, lightning/, etc.)
* Each functional package should be self-contained with models, parser, query, tests
* Use the fetch_data.sh script for getting test data from FMI API

## Weather Stations

* Focus on 16 coastal/maritime stations, especially Porkkala area
* Station IDs and details are documented in pkg/fmi/README.md  
* Test with stations 100996 (Harmaja), 101023 (Emäsalo), 151028 (Vuosaari)

