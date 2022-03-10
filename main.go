package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v38/github"
	"golang.org/x/oauth2"
)

var (
	baseCovLinkTemplate = flag.String("link-template.base", "", "template used to generate links to base commit coverage profiles. See LINK TEMPLATES")
	headCovLinkTemplate = flag.String("link-template.head", "", "template used to generate links to head commit coverage profiles. See LINK TEMPLATES")

	dryrun = flag.Bool("dry-run", false, "skip posting comment to Github, printing to stdout instead")
)

func init() {
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "Usage: %s [OPTIONS] <base-profile> <head-profile>\n", os.Args[0])

		fmt.Fprintln(out, "\nOPTIONS")
		flag.PrintDefaults()

		fmt.Fprintln(out, "\nLINK TEMPLATES")
		fmt.Fprintln(out, "In the comment posted to Github PR can optionally include links to")
		fmt.Fprintln(out, "the html version of the cover profile with color coded lines. In")
		fmt.Fprintln(out, "order to generate these links pass two links templates. The")
		fmt.Fprintln(out, "following tokens will be replaced in the template:")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "\t%[p] -> the current go package name")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Example")
		fmt.Fprintln(out, "$ goverdiff \\")
		fmt.Fprintln(out, "\t-link-template.base 'https://ci.acme.com/coverprofiles/before/%[p]/coverage.html' \\")
		fmt.Fprintln(out, "\t-link-template.head 'https://ci.acme.com/coverprofiles/after/%[p]/coverage.html' \\")
		fmt.Fprintln(out, "\t base/coverage.out head/coverage.out")
	}
}

func main() {
	ctx := context.Background()
	flag.Parse()

	if *baseCovLinkTemplate != "" {
		if *headCovLinkTemplate == "" {
			fmt.Fprintln(os.Stderr, "fatal: expected link-template.head option to not be empty")
			os.Exit(1)
		}
	} else if *headCovLinkTemplate != "" {
		fmt.Fprintln(os.Stderr, "fatal: expected link-template.base option to not be empty")
		os.Exit(1)
	}

	// load given base and head `go test` cover profiles from disk
	base, err := LoadCoverProfile(flag.Arg(0))
	if err != nil {
		panic(err)
	}

	head, err := LoadCoverProfile(flag.Arg(1))
	if err != nil {
		panic(err)
	}

	// generate and publish GitHub pull request message
	sum := summaryMessage(base.Coverage(), head.Coverage())
	tab := buildTable(getModulePackageName(), base, head)

	if *dryrun {
		fmt.Println(sum)
		fmt.Println(tab)
	} else {
		createOrUpdateComment(ctx, sum, tab)
	}
}

func buildTable(rootPkgName string, base, head *CoverProfile) string {
	const tableRowSprintf = "|%-80s|%8s|%8s|%8s|\n"
	rootPkgName += "/"

	// write report header
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf(tableRowSprintf, "package", "before", "after", "delta"))
	buf.WriteString(fmt.Sprintf(tableRowSprintf, "-------", "------", "-----", "-----"))

	// write package lines
	for _, pkgName := range getAllPackages(base, head) {
		baseCov := base.Packages[pkgName].Coverage()
		headCov := head.Packages[pkgName].Coverage()
		relPkg := relativePackage(rootPkgName, pkgName)
		templateReplacer := strings.NewReplacer("%[p]", relPkg)
		buf.WriteString(fmt.Sprintf(tableRowSprintf,
			relPkg,
			renderLinkTemplate(*baseCovLinkTemplate, coverageDescription(baseCov), templateReplacer),
			renderLinkTemplate(*headCovLinkTemplate, coverageDescription(headCov), templateReplacer),
			diffDescription(baseCov, headCov)))
	}

	// write totals
	buf.WriteString(fmt.Sprintf("|%80s|%8s|%8s|%8s|\n",
		"total:",
		coverageDescription(base.Coverage()),
		coverageDescription(head.Coverage()),
		diffDescription(base.Coverage(), head.Coverage()),
	))

	return buf.String()
}

func createOrUpdateComment(ctx context.Context, title, details string) {
	const coverageReportHeaderMarkdown = "# Golang test coverage diff report"

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
	tc := oauth2.NewClient(context.Background(), ts)

	client := github.NewClient(tc)
	comments, _, err := client.Issues.ListComments(ctx, owner, repo, prNum, &github.IssueListCommentsOptions{})
	if err != nil {
		panic(err)
	}

	// iterate over existing pull request comments - if existing coverage comment found then update
	body := fmt.Sprintf("%s\n%s\n\n\n%s\n",
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
	_, _, err = client.Issues.CreateComment(ctx, owner, repo, prNum, &github.IssueComment{
		Body: &body,
	})
	if err != nil {
		panic(err)
	}
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

func diffDescription(base, head int) string {
	if base < 0 && head < 0 {
		return "n/a"
	}
	if base < 0 {
		return "new"
	}
	if head < 0 {
		return "gone"
	}

	return fmt.Sprintf("%+6.2f%%", float64(head-base)/100)
}

func summaryMessage(base, head int) string {
	if base == head {
		return "Coverage unchanged."
	}

	if base > head {
		return fmt.Sprintf("Coverage decreased by `%.2f%%`. :bell: Shame :bell:", float64(base-head)/100)
	}

	return fmt.Sprintf("Coverage increased by `%.2f%%`. :medal_sports: Keep it up :medal_sports:", float64(head-base)/100)
}

func getModulePackageName() string {
	f, err := os.ReadFile("go.mod")
	if err != nil {
		// unable to determine package name
		return ""
	}

	// found it, stop searching
	modRegex := regexp.MustCompile(`module +([^\s]+)`)
	return string(modRegex.FindSubmatch(f)[1])
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

func renderLinkTemplate(tmpl, text string, rep *strings.Replacer) string {
	if tmpl == "" {
		return text
	}
	return fmt.Sprintf("[%s](%s)", text, rep.Replace(tmpl))
}
