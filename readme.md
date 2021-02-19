### goverdiff

This tool will compares two coverprofiles produced by go test and reports the deltas to github.

Example github actions workflow to integrate:

```yaml
name: Coverage report

on:
  pull_request:

jobs:
  cover:
    name: Coverage
    runs-on: ubuntu-latest
    steps:
      - name: Setup Golang
        uses: actions/setup-go@v2
        with:
          go-version: ~1.15
      - name: Checkout base
        uses: actions/checkout@v2
        with:
          ref: ${{ github.event.pull_request.base.ref }}
      - name: Cache base coverage data
        id: cache-base
        uses: actions/cache@v2
        with:
          path: base.profile
          key: ${{ hashFiles('**/*.go') }}
      - name: Extract base coverage
        if: steps.cache-base.outputs.cache-hit != 'true'
        run: go test -cover -coverprofile=base.profile ./...
      - name: Checkout head
        uses: actions/checkout@v2
        with:
          ref: ${{ github.event.pull_request.head.ref }}
          clean: false
      - name: Cache head coverage data
        id: cache-head
        uses: actions/cache@v2
        with:
          path: head.profile
          key: ${{ hashFiles('**/*.go') }}
      - name: Extract head coverage
        if: steps.cache-head.outputs.cache-hit != 'true'
        run: go test -cover -coverprofile=head.profile ./...
      - name: Diff coverage
        run: |
          GO111MODULE=off go get -u github.com/flipgroup/goverdiff && \
            goverdiff base.profile head.profile
        env:
          GITHUB_PULL_REQUEST_ID: ${{ github.event.number }}
          GITHUB_TOKEN: ${{ github.token }}
```

Note: This only works on pull_request. A push doesnt have a base ref to check against.
