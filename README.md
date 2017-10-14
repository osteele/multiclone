# Multiclone

Clone all the forks of a repository. Useful for collecting and reviewing
assignments and student projects.

## Usage

    multiclone https://github.com/owner/repo [directory]
    multiclone owner/repo [directory]
    multiclone --help

## Install

1. **Install go** (1) via [Homebrew](https://brew.sh): `brew install go`; or (2) [download](https://golang.org/doc/install#tarball).
2. `go install github.com/osteele/multiclone`
3. Create a [GitHub personal access token for the command line](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/)
4. Set `GITHUB_TOKEN` to this value: `export GITHUB_TOKEN=…`

## License

MIT
