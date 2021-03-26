### goverdiff

This tool will compare two cover profiles produced by `go test` and reports deltas back into a GitHub pull request.

Example GitHub Actions workflow to integrate:

```yaml
name: Coverage report

on:
  pull_request:

jobs:
  main:
    name: Coverage
    runs-on: ubuntu-latest
    steps:
      - name: Setup Golang
        uses: actions/setup-go@v2
        with:
          go-version: ~1.16
      - name: Checkout base
        uses: actions/checkout@v2
        with:
          ref: ${{ github.event.pull_request.base.ref }}
      - name: Cache base test coverage data
        id: cache-base
        uses: actions/cache@v2
        with:
          path: base.profile
          key: ${{ hashFiles('**/*.go') }}
      - name: Extract base test coverage
        if: steps.cache-base.outputs.cache-hit != 'true'
        run: go test -cover -coverprofile=base.profile ./...
      - name: Checkout head
        uses: actions/checkout@v2
        with:
          ref: ${{ github.event.pull_request.head.ref }}
          clean: false
      - name: Cache head test coverage data
        id: cache-head
        uses: actions/cache@v2
        with:
          path: head.profile
          key: ${{ hashFiles('**/*.go') }}
      - name: Extract head test coverage
        if: steps.cache-head.outputs.cache-hit != 'true'
        run: go test -cover -coverprofile=head.profile ./...
      - name: Diff test coverage
        env:
          GITHUB_PULL_REQUEST_ID: ${{ github.event.number }}
          GITHUB_TOKEN: ${{ github.token }}
        run: |
          go install github.com/flipgroup/goverdiff@main && \
            goverdiff base.profile head.profile
```

**Note:** This only works for a `pull_request` workflow event type. A push doesn't provide a `base.ref` to check against.
