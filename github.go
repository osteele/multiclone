package main

import (
	"context"
	"os"
	"strings"

	"github.com/shurcooL/githubql"
	"golang.org/x/oauth2"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type GitHubClient struct {
	c *githubql.Client
}

type repoRecord struct {
	Name  string
	Owner string
	URL   string
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

func newClient() (GitHubClient, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		kingpin.Errorf("Set GITHUB_TOKEN to a personal access token https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/")
	}
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	return GitHubClient{githubql.NewClient(httpClient)}, nil
}

func (c GitHubClient) queryOrgRepos(owner, name string) ([]repoRecord, error) {
	client := c.c
	variables := map[string]interface{}{
		"owner":          githubql.String(owner),
		"commentsCursor": (*githubql.String)(nil),
	}
	var repos []repoRecord
	hasNextPage := true
	for hasNextPage {
		if err := client.Query(context.Background(), &orgReposQuery, variables); err != nil {
			return nil, err
		}
		for _, repo := range orgReposQuery.Organization.Repositories.Nodes {
			if strings.HasPrefix(string(repo.Name), name+"-") {
				repos = append(repos, repoRecord{Name: string(repo.Name), Owner: string(repo.Owner.Login), URL: string(repo.URL)})
			}
		}
		variables["commentsCursor"] = githubql.NewString(orgReposQuery.Organization.Repositories.PageInfo.EndCursor)
		hasNextPage = bool(orgReposQuery.Organization.Repositories.PageInfo.HasNextPage)
	}
	return repos, nil
}

func (c GitHubClient) queryRepoForks(owner, name string) ([]repoRecord, error) {
	client := c.c
	variables := map[string]interface{}{
		"owner":          githubql.String(owner),
		"name":           githubql.String(name),
		"commentsCursor": (*githubql.String)(nil),
	}

	var repos []repoRecord
	hasNextPage := true
	for hasNextPage {
		if err := client.Query(context.Background(), &repoForksQuery, variables); err != nil {
			return nil, err
		}
		for _, repo := range repoForksQuery.Repository.Forks.Nodes {
			repos = append(repos, repoRecord{Name: string(repo.Name), Owner: string(repo.Owner.Login), URL: string(repo.URL)})
		}
		variables["commentsCursor"] = githubql.NewString(repoForksQuery.Repository.Forks.PageInfo.EndCursor)
		hasNextPage = bool(repoForksQuery.Repository.Forks.PageInfo.HasNextPage)
	}
	return repos, nil
}
