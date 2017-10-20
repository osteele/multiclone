package main

import (
	"context"
	"os"
	"strings"

	"github.com/shurcooL/githubql"
	"golang.org/x/oauth2"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

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
