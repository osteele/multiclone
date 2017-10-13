package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/shurcooL/githubql"
	"golang.org/x/oauth2"
)

var (
	dry_run = kingpin.Flag("dry-run", "Dry Run").Bool()
	jobs    = kingpin.Flag("jobs", "The number of repos fetched at the same time").Short('j').Default("8").Int()
	nwo     = kingpin.Arg("repo", "GitHub owner/repo").String()
	dir     = kingpin.Arg("directory", "The name of the directory to clone into.").Default(".").String()

	repo_re = regexp.MustCompile(`^(?:https://github\.com/)?([^/]+)/([^/]+)$`)
)

func main() {
	kingpin.Parse()
	if *nwo == "" {
		kingpin.FatalUsage("repo is a required argument")
	}
	m := repo_re.FindStringSubmatch(*nwo)
	if m == nil {
		kingpin.FatalUsage("repo must be in the format owner/repo")
	}
	if err := run(m[1], m[2], *dir); err != nil {
		kingpin.FatalIfError(err, "")
	}
}

type repoForksNode struct {
	URL   githubql.String
	Owner struct {
		Login githubql.String
	}
}

var repoForksQuery struct {
	// https://developer.github.com/v4/reference/object/repository/
	Repository struct {
		Description githubql.String
		Forks       struct {
			Nodes    []repoForksNode
			PageInfo struct {
				EndCursor   githubql.String
				HasNextPage githubql.Boolean
			}
		} `graphql:"forks(first: 20, after: $commentsCursor)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

func queryForks(owner, name string) ([]repoForksNode, error) {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	client := githubql.NewClient(httpClient)
	variables := map[string]interface{}{
		"owner":          githubql.String(owner),
		"name":           githubql.String(name),
		"commentsCursor": (*githubql.String)(nil),
	}

	var repos []repoForksNode
	hasNextPage := true
	for hasNextPage {
		if err := client.Query(context.Background(), &repoForksQuery, variables); err != nil {
			return nil, err
		}
		repos = append(repos, repoForksQuery.Repository.Forks.Nodes...)
		variables["commentsCursor"] = githubql.NewString(repoForksQuery.Repository.Forks.PageInfo.EndCursor)
		hasNextPage = bool(repoForksQuery.Repository.Forks.PageInfo.HasNextPage)
	}

	return repos, nil
}

func run(owner, name, dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	results := make(chan []byte, *jobs)
	repos, err := queryForks(owner, name)
	if err != nil {
		return err
	}
	for _, repo := range repos {
		dst := filepath.Join(dir, string(repo.Owner.Login))
		go func(url, dst string) {
			args := []string{"git", "clone", url, dst}
			if *dry_run {
				args = append([]string{"echo"}, args...)
			}
			cmd := exec.Command(args[0], args[1:]...)
			stdoutStderr, err := cmd.CombinedOutput()
			if err != nil {
				// FIXME send to main thread
				log.Fatal(err)
			}
			results <- bytes.TrimSpace(stdoutStderr)
		}(string(repo.URL), dst)
	}

	for n := len(repos); n > 0; n-- {
		fmt.Printf("%s\n", <-results)
	}
	return nil
}
