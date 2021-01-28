package main

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
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

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%-84s %7s %7s %8s\n", "package name", "before", "after", "delta"))
	for _, pkg := range getAllPackages(base, head) {
		baseCov := base.Packages[pkg].Coverage()
		headCov := head.Packages[pkg].Coverage()
		buf.WriteString(fmt.Sprintf("%-84s %7s %7s %8s\n", pkg, coverageDescription(baseCov), coverageDescription(headCov), diffDescription(baseCov, headCov)))
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
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("no GITHUB_TOKEN, not reporting to github.")
		return
	}
	ownerAndRepo := os.Getenv("GITHUB_REPOSITORY")
	if ownerAndRepo == "" {
		fmt.Println("no GITHUB_REPOSITORY, not reporting to github.")
		return
	}
	repoparts := strings.SplitN(ownerAndRepo, "/", 2)
	owner := repoparts[0]
	repo := repoparts[1]

	prIdStr := os.Getenv("GITHUB_PULL_REQUEST_ID")
	if prIdStr == "" {
		fmt.Println("no GITHUB_PULL_REQUEST_ID, not reporting to github.")
		return
	}
	prID, err := strconv.Atoi(os.Getenv("GITHUB_PULL_REQUEST_ID"))
	if err != nil {
		fmt.Println("GITHUB_PULL_REQUEST_ID not a valid number, not reporting to github.")
		return
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	comments, _, err := client.Issues.ListComments(ctx, owner, repo, prID, &github.IssueListCommentsOptions{})
	if err != nil {
		panic(err)
	}

	body := "### coverage diff\n" + title + "\n\n```\n" + details + "```\n"

	for _, c := range comments {
		if c.Body == nil {
			continue
		}

		if *c.Body == body {
			// no change required, lets GTFO.
			return
		}

		if strings.HasPrefix(*c.Body, "### coverage diff") {
			_, _, err = client.Issues.EditComment(ctx, owner, repo, *c.ID, &github.IssueComment{
				Body: &body,
			})
			if err != nil {
				panic(err)
			}
			return
		}
	}
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
	if math.IsNaN(base) {
		return "new"
	}
	if math.IsNaN(head) {
		return "deleted"
	}

	return fmt.Sprintf("%+6.1f%%", (head-base)*100)
}

func getAllPackages(profiles ...*CoverProfile) []string {
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
