# Multiclone

Clone all the forks of a repository, or all the repos of a [GitHub Classroom](https://classroom.github.com) assignment.

Repos are cloned in parallel.

This is useful for collecting and reviewing
assignments and student projects.

## Usage

    multiclone https://github.com/owner/repo [DIR]
    multiclone owner/repo [DIR]

Clone forks of owner/repo into DIR (or the current directory).

### GitHub Classroom

    multiclone https://github.com/owner/repo [DIR] --classroom
    multiclone org/repo [DIR] --classroom

Clone org's repos named repo-* into DIR (or the current directory).

This is intended for use with repos created via [GitHub Classroom](https://classroom.github.com).

### Usage

    multiclone --help

## Install

1. **Install go** (1) via [Homebrew](https://brew.sh): `brew install go`; or (2) [download](https://golang.org/doc/install#tarball).
2. `go install github.com/osteele/multiclone`
3. Create a [GitHub personal access token for the command line](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/)
4. Set `GITHUB_TOKEN` to this value: `export GITHUB_TOKEN=…`

## License

MIT
