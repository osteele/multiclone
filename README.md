# Multiclone

Clone all the forks of a repository. Useful for collecting and reviewing
assignments and student projects.

## Usage

Create a [GitHub personal access token for the command line](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/),
and set `GITHUB_TOKEN` to this value.

    export GITHUB_TOKEN=…
    multiclone https://github.com/owner/reponame [directory]
    multiclone owner/reponame [directory]
    multiclone --help

## Install

1. **Install go** (1) via [Homebrew](https://brew.sh): `brew install go`; or (2) [download](https://golang.org/doc/install#tarball).
2. `go get github.com/osteele/multiclone`

# License

MIT
