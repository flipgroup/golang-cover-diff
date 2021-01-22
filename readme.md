### goverdiff

This tool will compares two coverprofiles produced by go test and reports the deltas to github.

Example github actions workflow to integrate:
```
name: Cover

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
      - name: Extract base coverage
        run: make setup-test && go test -cover -coverprofile=base.profile ./...
      - name: Checkout head
        uses: actions/checkout@v2
        with:
          ref: ${{ github.event.pull_request.head.ref }}
          clean: false
      - name: Extract head coverage
        run: go test -cover -coverprofile=head.profile ./...
      - name: Diff coverage
        run: GO111MODULE=off go get -u github.com/flipgroup/goverdiff && goverdiff base.profile head.profile
        env:
          GITHUB_TOKEN:  ${{ github.token }}
          GITHUB_PULL_REQUEST_ID: ${{ github.event.number }}
```

Note: This only works on pull_request. A push doesnt have a base ref to check against.