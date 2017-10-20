package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"text/template"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
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
