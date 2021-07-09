package main

import (
	"bytes"
	"context"
	"fmt"
	"math"
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
	base, err := LoadCoverProfile(os.Args[1])
	if err != nil {
		panic(err)
	}

	head, err := LoadCoverProfile(os.Args[2])
	if err != nil {
		panic(err)
	}

	rootName := getModulePackageName()

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%-84s %7s %7s %8s\n", "package name", "before", "after", "delta"))
	for _, pkg := range getAllPackages(base, head) {
		baseCov := base.Packages[pkg].Coverage()
		headCov := head.Packages[pkg].Coverage()
		buf.WriteString(fmt.Sprintf("%-84s %7s %7s %8s\n", relativePackage(pkg, rootName), coverageDescription(baseCov), coverageDescription(headCov), diffDescription(baseCov, headCov)))
	}
	buf.WriteString(fmt.Sprintf("%84s %7s %7s %8s\n", "total:", coverageDescription(base.Coverage()), coverageDescription(head.Coverage()), diffDescription(base.Coverage(), head.Coverage())))

	fmt.Println(buf.String())

	var title string
	if base.Coverage() == head.Coverage() {
		title = "Coverage unchanged."
	} else if base.Coverage() > head.Coverage() {
		title = fmt.Sprintf("Coverage decreased by %6.1f%%. :bell: Shame :bell:", (base.Coverage()-head.Coverage())*100)
	} else {
		title = fmt.Sprintf("Coverage increased by %6.1f%%. Keep it up :medal_sports:", (head.Coverage()-base.Coverage())*100)
	}

	createOrUpdateComment(context.Background(), title, buf.String())
}

func createOrUpdateComment(ctx context.Context, title string, details string) {
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
	body := fmt.Sprintf("### coverage diff\n%s\n\n```\n%s```\n", title, details)
	for _, c := range comments {
		if c.Body == nil {
			continue
		}

		if *c.Body == body {
			// existing comment body is the same - no change required
			return
		}

		if strings.HasPrefix(*c.Body, "### coverage diff") {
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

func coverageDescription(coverage float64) string {
	if math.IsNaN(coverage) {
		return "-"
	}
	return fmt.Sprintf("%6.1f%%", coverage*100)
}

func diffDescription(base float64, head float64) string {
	if math.IsNaN(base) && math.IsNaN(head) {
		return ""
	}
	if math.IsNaN(base) {
		return "new"
	}
	if math.IsNaN(head) {
		return "deleted"
	}

	return fmt.Sprintf("%+6.1f%%", (head-base)*100)
}

func relativePackage(pkg string, root string) string {
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
	return string(modregex.FindSubmatch(f)[1]) + "/"
}

var modregex = regexp.MustCompile(`module ([^\s]*)`)

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
