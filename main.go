package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/shurcooL/githubql"
	"golang.org/x/oauth2"
)

var (
	dry_run = kingpin.Flag("dry-run", "Dry Run").Bool()
	jobs    = kingpin.Flag("jobs", "The number of repos fetched at the same time").Short('j').Default("8").Int()
	nwo     = kingpin.Arg("repo", "GitHub owner/repo").String()
	dir     = kingpin.Arg("directory", "The name of the directory to clone into.").Default(".").String()
)

func main() {
	kingpin.Parse()
	parts := strings.Split(*nwo, "/")
	if len(parts) != 2 {
		kingpin.FatalUsage("repo must be in the format owner/repo")
	}
	if err := run(parts[0], parts[1], *dir); err != nil {
		panic(err)
	}
}

type repoNode struct {
	URL   githubql.String
	Owner struct {
		Login githubql.String
	}
}

var repoQuery struct {
	// https://developer.github.com/v4/reference/object/repository/
	Repository struct {
		Description githubql.String
		Forks       struct {
			Nodes    []repoNode
			PageInfo struct {
				EndCursor   githubql.String
				HasNextPage githubql.Boolean
			}
		} `graphql:"forks(first: 2, after: $commentsCursor)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

func queryForks(owner, name string) ([]repoNode, error) {
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

	var repos []repoNode
	hasNextPage := true
	for hasNextPage {
		if err := client.Query(context.Background(), &repoQuery, variables); err != nil {
			return nil, err
		}
		repos = append(repos, repoQuery.Repository.Forks.Nodes...)
		variables["commentsCursor"] = githubql.NewString(repoQuery.Repository.Forks.PageInfo.EndCursor)
		hasNextPage = bool(repoQuery.Repository.Forks.PageInfo.HasNextPage)
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