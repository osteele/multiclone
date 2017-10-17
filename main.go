package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/shurcooL/githubql"
	"golang.org/x/oauth2"
)

var (
	dryRun    = kingpin.Flag("dry-run", "Dry run").Bool()
	classroom = kingpin.Flag("classroom", "Repo is GitHub classroom repo").Bool()
	jobs      = kingpin.Flag("jobs", "The number of repos fetched at the same time").Short('j').Default("8").Int()
	mrconfig  = kingpin.Flag("mrconfig", "Create a myrepos .mrconfig file in the output directory").Default("true").Bool()
	nwo       = kingpin.Arg("repo", "GitHub owner/repo").String()
	dir       = kingpin.Arg("directory", "The name of the directory to clone into").Default(".").String()

	repoRE = regexp.MustCompile(`^(?:https://github\.com/)?([^/]+)/([^/]+)$`)
)

func main() {
	kingpin.Parse()
	if *nwo == "" {
		kingpin.FatalUsage("repo is a required argument")
	}
	m := repoRE.FindStringSubmatch(*nwo)
	if m == nil {
		kingpin.FatalUsage("repo must be in the format owner/repo")
	}
	owner, name := m[1], m[2]
	if err := run(owner, name); err != nil {
		kingpin.FatalIfError(err, "")
	}
}

func run(owner, name string) error {
	repos, err := queryRepos(owner, name)
	if err != nil {
		return err
	}
	if err := cloneRepos(repos, name, *dir); err != nil {
		return err
	}
	if *mrconfig {
		return writeMrConfig(repos, name, *dir)
	}
	return nil
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

func repoLocalBasename(repo repoNode, name string) string {
	if *classroom {
		return string(repo.Name)[len(name)+1:]
	}
	return string(repo.Owner.Login)
}

func cloneRepos(repos []repoNode, name, dir string) error {
	if !*dryRun {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	var (
		sem     = make(chan bool, *jobs)
		errors  = make(chan error, 1)
		outputs = make(chan []byte, 1)
	)
	for _, repo := range repos {
		dst := filepath.Join(dir, repoLocalBasename(repo, name))
		go func(url, dst string) {
			sem <- true
			defer func() { <-sem }()
			args := []string{"git", "clone", url, dst}
			if *dryRun {
				args = append([]string{"echo"}, args...)
				// time.Sleep(time.Second)
			}
			cmd := exec.Command(args[0], args[1:]...)
			stdoutStderr, err := cmd.CombinedOutput()
			if err != nil {
				errors <- fmt.Errorf("%s: %s while trying to clone %s", err, stdoutStderr, url)
			} else {
				outputs <- bytes.TrimSpace(stdoutStderr)
			}
		}(string(repo.URL), dst)
	}
	errorCount := 0
	for n := len(repos); n > 0; {
		select {
		case output := <-outputs:
			n--
			fmt.Printf("%s\n", output)
		case err := <-errors:
			fmt.Fprintf(os.Stderr, "%s\n", err)
			n--
			errorCount++
		}
	}
	if errorCount > 0 {
		return fmt.Errorf("one or more clones failed")
	}
	return nil
}

func writeMrConfig(repos []repoNode, name, dir string) error {
	dst := filepath.Join(dir, ".mrconfig")
	if *dryRun {
		fmt.Println("writing", dst)
		return nil
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	type RepoEntry struct {
		Dir, URL string
	}
	var repoEntries []RepoEntry
	for _, repo := range repos {
		repoEntries = append(repoEntries, RepoEntry{Dir: repoLocalBasename(repo, name), URL: string(repo.URL)})
	}
	return mrConfigTpl.Execute(f, repoEntries)
}

var mrConfigTpl = template.Must(template.New("mrconfig").Parse(`
{{- range . -}}
[{{ .Dir }}]
checkout = git clone {{ .URL }} {{ .Dir }}

{{ end -}}
`))
