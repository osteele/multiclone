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
	"strings"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/shurcooL/githubql"
	"golang.org/x/oauth2"
)

var (
	dry_run   = kingpin.Flag("dry-run", "Dry run").Bool()
	jobs      = kingpin.Flag("jobs", "The number of repos fetched at the same time").Short('j').Default("8").Int()
	classroom = kingpin.Flag("classroom", "Repo is GitHub classroom repo").Bool()
	nwo       = kingpin.Arg("repo", "GitHub owner/repo").String()
	dir       = kingpin.Arg("directory", "The name of the directory to clone into.").Default(".").String()

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
	owner, name := m[1], m[2]
	repos, err := queryRepos(owner, name)
	if err != nil {
		kingpin.FatalIfError(err, "")
	}
	if err := cloneRepos(repos, name, *dir); err != nil {
		kingpin.FatalIfError(err, "")
	}
}

type repoNode struct {
	Name  githubql.String
	URL   githubql.String
	Owner struct {
		Login githubql.String
	}
}

var repoForksQuery struct {
	Repository struct {
		Forks struct {
			Nodes    []repoNode
			PageInfo struct {
				EndCursor   githubql.String
				HasNextPage githubql.Boolean
			}
		} `graphql:"forks(first: 20, after: $commentsCursor)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

var orgReposQuery struct {
	Organization struct {
		Repositories struct {
			Nodes    []repoNode
			PageInfo struct {
				EndCursor   githubql.String
				HasNextPage githubql.Boolean
			}
		} `graphql:"repositories(first: 20, after: $commentsCursor)"`
	} `graphql:"organization(login: $owner)"`
}

func newClient() (*githubql.Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		kingpin.Errorf("Set GITHUB_TOKEN to a personal access token https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/")
	}
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	return githubql.NewClient(httpClient), nil
}

func queryOrgRepos(owner, name string) ([]repoNode, error) {
	client, err := newClient()
	if err != nil {
		return nil, err
	}
	variables := map[string]interface{}{
		"owner":          githubql.String(owner),
		"commentsCursor": (*githubql.String)(nil),
	}
	var repos []repoNode
	hasNextPage := true
	for hasNextPage {
		if err := client.Query(context.Background(), &orgReposQuery, variables); err != nil {
			return nil, err
		}
		for _, repo := range orgReposQuery.Organization.Repositories.Nodes {
			if strings.HasPrefix(string(repo.Name), name+"-") {
				repos = append(repos, repo)
			}
		}
		variables["commentsCursor"] = githubql.NewString(orgReposQuery.Organization.Repositories.PageInfo.EndCursor)
		hasNextPage = bool(orgReposQuery.Organization.Repositories.PageInfo.HasNextPage)
	}
	return repos, nil
}

func queryRepoForks(owner, name string) ([]repoNode, error) {
	client, err := newClient()
	if err != nil {
		return nil, err
	}
	variables := map[string]interface{}{
		"owner":          githubql.String(owner),
		"name":           githubql.String(name),
		"commentsCursor": (*githubql.String)(nil),
	}

	var repos []repoNode
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

func queryRepos(owner, name string) ([]repoNode, error) {
	if *classroom {
		return queryOrgRepos(owner, name)
	}
	return queryRepoForks(owner, name)
}

func cloneRepos(repos []repoNode, name, dir string) error {
	if !*dry_run {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	results := make(chan []byte, *jobs)
	for _, repo := range repos {
		dst := filepath.Join(dir, string(repo.Owner.Login))
		if *classroom {
			dst = filepath.Join(dir, dst, string(repo.Name)[len(name)+1:])
		}
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
