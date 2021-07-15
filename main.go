package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v34/github"
	"golang.org/x/oauth2"
	"golang.org/x/tools/go/packages"
)

func main() {
	// load given base and head `go test` cover profiles from disk
	base, err := LoadCoverProfile(os.Args[1])
	if err != nil {
		panic(err)
	}

	head, err := LoadCoverProfile(os.Args[2])
	if err != nil {
		panic(err)
	}

	rootName := getModulePackageName()

	// write report header
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("%-84s %7s %7s %8s\n", "package name", "before", "after", "delta"))

	// write package lines
	for _, pkg := range getAllPackages(base, head) {
		baseCov := base.Packages[pkg].Coverage()
		headCov := head.Packages[pkg].Coverage()
		buf.WriteString(fmt.Sprintf("%-84s %7s %7s %8s\n",
			relativePackage(pkg, rootName),
			coverageDescription(baseCov),
			coverageDescription(headCov),
			diffDescription(baseCov, headCov)))
	}

	// write totals
	buf.WriteString(fmt.Sprintf("%84s %7s %7s %8s\n", "total:",
		coverageDescription(base.Coverage()),
		coverageDescription(head.Coverage()),
		diffDescription(base.Coverage(), head.Coverage()),
	))

	// generate GitHub pull request message
	createOrUpdateComment(
		context.Background(),
		summaryMessage(base.Coverage(), head.Coverage()),
		buf.String())
}

func createOrUpdateComment(ctx context.Context, title, details string) {
	const coverageReportHeaderMarkdown = "### coverage diff"

	auth_token := os.Getenv("GITHUB_TOKEN")
	if auth_token == "" {
		fmt.Println("no GITHUB_TOKEN, unable to report back to GitHub pull request.")
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

	prIdStr := os.Getenv("GITHUB_PULL_REQUEST_ID")
	if prIdStr == "" {
		fmt.Println("no GITHUB_PULL_REQUEST_ID, not reporting to GitHub.")
		return
	}

	prID, err := strconv.Atoi(os.Getenv("GITHUB_PULL_REQUEST_ID"))
	if err != nil {
		fmt.Println("provided GITHUB_PULL_REQUEST_ID is not a valid number, not reporting to GitHub.")
		return
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: auth_token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	client := github.NewClient(tc)
	comments, _, err := client.Issues.ListComments(ctx, owner, repo, prID, &github.IssueListCommentsOptions{})
	if err != nil {
		panic(err)
	}

	// iterate over existing pull request comments - if existing coverage comment found then update
	body := fmt.Sprintf("%s\n%s\n\n```\n%s```\n",
		coverageReportHeaderMarkdown,
		title, details)

	for _, c := range comments {
		if c.Body == nil {
			continue
		}

		if *c.Body == body {
			// existing comment body is the same - no change required
			return
		}

		if strings.HasPrefix(*c.Body, coverageReportHeaderMarkdown) {
			// found existing cover comment - update
			_, _, err = client.Issues.EditComment(ctx, owner, repo, *c.ID, &github.IssueComment{
				Body: &body,
			})
			if err != nil {
				panic(err)
			}
			return
		}
	}

	// no coverage comment found - create new comment
	_, _, err = client.Issues.CreateComment(ctx, owner, repo, prID, &github.IssueComment{
		Body: &body,
	})
	if err != nil {
		panic(err)
	}
}

func coverageDescription(coverage int) string {
	if coverage < 0 {
		return "-"
	}
	return fmt.Sprintf("%6.2f%%", float64(coverage)/100)
}

func diffDescription(base, head int) string {
	if base < 0 && head < 0 {
		return ""
	}
	if base < 0 {
		return "new"
	}
	if head < 0 {
		return "deleted"
	}

	return fmt.Sprintf("%+6.2f%%", float64(head-base)/100)
}

func summaryMessage(base, head int) string {
	if base == head {
		return "Coverage unchanged."
	}

	if base > head {
		return fmt.Sprintf("Coverage decreased by %6.1f%%. :bell: Shame :bell:", float64(base-head)/100)
	}

	return fmt.Sprintf("Coverage increased by %6.1f%%. Keep it up :medal_sports:", float64(head-base)/100)
}

func relativePackage(pkg, root string) string {
	if strings.HasPrefix(pkg, root) {
		return "./" + strings.TrimPrefix(pkg, root)
	}
	return pkg
}

func getModulePackageName() string {
	f, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}
	// found it, stop searching
	return string(modRegex.FindSubmatch(f)[1]) + "/"
}

var modRegex = regexp.MustCompile(`module ([^\s]*)`)

func getAllPackages(profiles ...*CoverProfile) []string {
	set := map[string]struct{}{}

	pkg, err := packages.Load(&packages.Config{Mode: packages.NeedName}, "./...")
	if err != nil {
		panic(err)
	}
	for _, p := range pkg {
		set[p.PkgPath] = struct{}{}
	}

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
