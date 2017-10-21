package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
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
	verbose   = kingpin.Flag("verbose", "Verbose").Bool()
	mrconfig  = kingpin.Flag("mrconfig", "Create a myrepos .mrconfig file in the output directory").Default("true").Bool()

	nwo = kingpin.Arg("repo", "GitHub owner/repo").String()
	dir = kingpin.Arg("directory", "The name of the directory to clone into").Default(".").String()

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

type repoEntry struct {
	Dir, URL string
}

func run(owner, name string) error {
	repos, err := queryRepos(owner, name)
	if err != nil {
		return err
	}
	if len(repos) == 0 {
		prep := "with"
		if *classroom {
			prep = "without"
		}
		fmt.Fprintf(os.Stderr, "No entries. Try again %s the --classroom option.\n", prep)
	}
	var entries []repoEntry
	for _, repo := range repos {
		entries = append(entries, repoEntry{Dir: repoAuthor(repo, name), URL: repo.URL})
	}
	if err := cloneRepos(entries, *dir); err != nil {
		return err
	}
	if *mrconfig {
		return writeMrConfig(entries, *dir)
	}
	return nil
}

func queryRepos(owner, name string) ([]repoRecord, error) {
	client, err := newClient()
	if err != nil {
		return nil, err
	}
	switch *classroom {
	case true:
		return client.queryOrgRepos(owner, name)
	default:
		return client.queryRepoForks(owner, name)
	}
}

func repoAuthor(repo repoRecord, name string) string {
	if *classroom {
		return repo.Name[len(name)+1:]
	}
	return repo.Owner
}

func cloneRepos(repos []repoEntry, dir string) error {
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
		go func(repo repoEntry) {
			sem <- true
			defer func() { <-sem }()
			// dst := filepath.Join(dir, repo.owner)
			args := []string{"git", "clone", repo.URL, repo.Dir}
			if *dryRun {
				args = append([]string{"echo"}, args...)
				// time.Sleep(time.Second)
			}
			cmd := exec.Command(args[0], args[1:]...)
			stdoutStderr, err := cmd.CombinedOutput()
			if err != nil {
				errors <- fmt.Errorf("%s: %s while trying to clone %s", err, stdoutStderr, repo.URL)
			} else {
				outputs <- bytes.TrimSpace(stdoutStderr)
			}
		}(repo)
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

func writeMrConfig(repos []repoEntry, dir string) error {
	dst := filepath.Join(dir, ".mrconfig")
	f := ioutil.Discard
	if *dryRun {
		fmt.Println("writing", dst)
	} else {
		f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer f.Close()
	}

	if *verbose {
		mrConfigTpl.Execute(os.Stdout, repos)
	}
	return mrConfigTpl.Execute(f, repos)
}

var mrConfigTpl = template.Must(template.New("mrconfig").Parse(`
{{- range . -}}
[{{ .Dir }}]
checkout = git clone {{ .URL }} {{ .Dir }}

{{ end -}}
`))
