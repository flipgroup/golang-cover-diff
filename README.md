# Golang coverage difference reporter

[![Test](https://github.com/flipgroup/golang-cover-diff/actions/workflows/test.yml/badge.svg)](https://github.com/flipgroup/golang-cover-diff/actions/workflows/test.yml)

This tool - designed for use within a GitHub Actions workflow, will compare `go test` produced cover profiles and report coverage deltas back into a GitHub pull request.

See [`cover.example.yml`](cover.example.yml) for an example coverage GitHub Actions workflow template.
