package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

func main() {
	ctx := context.Background()

	// load given base and head `go test` cover profiles from disk
	base, err := LoadCoverProfile(os.Args[1])
	if err != nil {
		panic(err)
	}

	head, err := LoadCoverProfile(os.Args[2])
	if err != nil {
		panic(err)
	}

	// generate and publish GitHub pull request message
	createOrUpdateComment(
		ctx,
		summaryMessage(base.Coverage(), head.Coverage()),
		buildTable(moduleName(), base, head))
}

func createOrUpdateComment(ctx context.Context, summary, reportTable string) {
	const commentMarker = "<!-- info:golang-cover-diff -->"

	auth_token := os.Getenv("GITHUB_TOKEN")
	if auth_token == "" {
		fmt.Println("no GITHUB_TOKEN, not reporting to GitHub.")
		return
	}

	ownerAndRepo := os.Getenv("GITHUB_REPOSITORY")
	if ownerAndRepo == "" {
		fmt.Println("no GITHUB_REPOSITORY, not reporting to GitHub.")
		return
	}

	parts := strings.SplitN(ownerAndRepo, "/", 2)
	owner := parts[0]
	repo := parts[1]

	prNumStr := os.Getenv("GITHUB_PULL_REQUEST_ID")
	if prNumStr == "" {
		fmt.Println("no GITHUB_PULL_REQUEST_ID, not reporting to GitHub.")
		return
	}

	prNum, err := strconv.Atoi(prNumStr)
	if err != nil {
		fmt.Println("provided GITHUB_PULL_REQUEST_ID is not a valid number, not reporting to GitHub.")
		return
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: auth_token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	comments, _, err := client.Issues.ListComments(ctx, owner, repo, prNum, &github.IssueListCommentsOptions{})
	if err != nil {
		panic(err)
	}

	// iterate over existing pull request comments - if existing coverage comment found then update
	body := buildCommentBody(commentMarker, summary, reportTable)
	for _, c := range comments {
		if c.Body == nil {
			continue
		}

		if *c.Body == body {
			// existing comment body is identical - no change
			return
		}

		if strings.HasPrefix(*c.Body, commentMarker) {
			// found existing coverage comment - update
			_, _, err := client.Issues.EditComment(ctx, owner, repo, *c.ID, &github.IssueComment{
				Body: &body,
			})
			if err != nil {
				panic(err)
			}
			return
		}
	}

	// no coverage comment found - create
	_, _, err = client.Issues.CreateComment(ctx, owner, repo, prNum, &github.IssueComment{
		Body: &body,
	})
	if err != nil {
		panic(err)
	}
}

func buildCommentBody(commentMarker, summary, reportTable string) string {
	return fmt.Sprintf(
		("%s\n" +
			"# Golang test coverage difference report\n\n" +
			"%s\n\n" +
			"<details>\n<summary>Package report</summary>\n\n" +
			"```\n%s\n```\n" +
			"</details>"),
		commentMarker,
		summary, reportTable)
}

func buildTable(rootPkgName string, base, head *CoverProfile) string {
	const tableRowSprintf = "%-80s  %7s  %7s  %7s\n"
	rootPkgName += "/"

	// write report header
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf(tableRowSprintf, "package", "before", "after", "delta"))
	buf.WriteString(fmt.Sprintf(tableRowSprintf, "-------", "-------", "-------", "-------"))

	// write package lines
	for _, pkgName := range allPackages(base, head) {
		baseCov := base.Packages[pkgName].Coverage()
		headCov := head.Packages[pkgName].Coverage()
		buf.WriteString(fmt.Sprintf(tableRowSprintf,
			relativePackage(rootPkgName, pkgName),
			coverageDescription(baseCov),
			coverageDescription(headCov),
			diffDescription(baseCov, headCov, true)))
	}

	// write totals
	buf.WriteString(fmt.Sprintf("%80s %8s %8s %8s",
		"total:",
		coverageDescription(base.Coverage()),
		coverageDescription(head.Coverage()),
		diffDescription(base.Coverage(), head.Coverage(), false),
	))

	return buf.String()
}

func relativePackage(root, pkgName string) string {
	pkgName = strings.TrimPrefix(pkgName, root)
	if len(pkgName) > 80 {
		pkgName = pkgName[:80]
	}

	return pkgName
}

func coverageDescription(coverage int) string {
	if coverage < 0 {
		return "-"
	}
	return fmt.Sprintf("%6.2f%%", float64(coverage)/100)
}

func diffDescription(base, head int, emptyNoDiff bool) string {
	if base < 0 && head < 0 {
		return "n/a"
	}
	if base < 0 {
		return "new"
	}
	if head < 0 {
		return "gone"
	}
	if base == head && emptyNoDiff {
		return ""
	}

	return fmt.Sprintf("%+6.2f%%", float64(head-base)/100)
}

func summaryMessage(base, head int) string {
	if base == head {
		return "Coverage unchanged. :2nd_place_medal:"
	}

	if base > head {
		return fmt.Sprintf("Coverage decreased by `%.2f%%`. :bell: Shame :bell:", float64(base-head)/100)
	}

	return fmt.Sprintf("Coverage increased by `%.2f%%`. :medal_sports: Keep it up :medal_sports:", float64(head-base)/100)
}

func moduleName() string {
	f, err := os.ReadFile("go.mod")
	if err != nil {
		// unable to determine package name
		return ""
	}

	// opened file - locate `module` line to extract full package name
	modRegex := regexp.MustCompile(`module +([^\s]+)`)
	return string(modRegex.FindSubmatch(f)[1])
}

func allPackages(profiles ...*CoverProfile) []string {
	set := map[string]struct{}{}
	for _, profile := range profiles {
		for name := range profile.Packages {
			set[name] = struct{}{}
		}
	}

	var res []string
	for name := range set {
		res = append(res, name)
	}

	// sort into stable order
	sort.Slice(res, func(i, j int) bool {
		return res[i] < res[j]
	})
	return res
}
