# Golang coverage difference reporter

[![Test](https://github.com/flipgroup/golang-cover-diff/actions/workflows/test.yml/badge.svg)](https://github.com/flipgroup/golang-cover-diff/actions/workflows/test.yml)

This tool is designed for use within a GitHub Actions workflow for comparing `go test` produced cover profiles and reporting coverage deltas back into a GitHub pull request as a comment entry.

See [`cover.example.yml`](cover.example.yml) for an example coverage GitHub Actions workflow template.
