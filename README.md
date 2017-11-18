# Multiclone

Clone all the forks of a repository, or all the repos of a [GitHub Classroom](https://classroom.github.com) assignment.

This is useful for collecting and reviewing assignments and student projects.

Features:

* Written in Golang for easier distribution. (I got tired juggling of juggling Anaconda / virtualenv between various classroom and tool environments.)
* Automatic repo discovery. Knows about “students fork” and “GitHub Classroom” conventions.
* Repos are cloned in parallel.
* Create a [myrepos](https://myrepos.branchable.com) `.mrconfig` file.

## Installation

### Install multiclone

multiclone is written in Go with support for multiple platforms.
The latest release can be found at the [releases page](https://github.com/osteele/multiclone/releases).

[Homebrew](https://brew.sh) can be used to install multiclone on macOS:

```bash
$ brew tap osteele/homebrew-tap
$ brew install multiclone
```

### Set `GITHUB_TOKEN`

Create a [GitHub personal access token for the command line](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/)

Set `GITHUB_TOKEN` to this value: `export GITHUB_TOKEN=…`

## Usage

    multiclone https://github.com/owner/repo
    multiclone owner/repo

Clone forks of owner/repo into the current directory.

    multiclone repos.txt

Clones the repo

`repos.txt` is a file with one repo name-with-owner, e.g. `osteele/homework1`, per line.

### GitHub Classroom

    multiclone https://github.com/owner/repo --classroom
    multiclone org/repo --classroom

Clone org's repos named repo-* into the current directory.

This is intended for use with repos created via [GitHub Classroom](https://classroom.github.com).

### Options

    multiclone --dir path/to/dir owner/repo

Clone into subdirectories of `path/to/dir`, instead of the current directory.

    multiclone owner/repo --dry-run

See the `git` commands that would be run, without actually running them.

    multiclone --help

Lists additional options.

## Develop

1. **Install go** (1) via [Homebrew](https://brew.sh): `brew install go`; or (2) [download](https://golang.org/doc/install#tarball).
2. `go install github.com/osteele/multiclone`

## Alternatives

These [GitHub Education Community](https://education.github.community/t/how-to-automatically-gather-or-collect-assignments/2595) forum threads discuss a variety of alternatives (including one I wrote before I wrote this):

* [GitHub Classroom: clone assignments](https://education.github.community/t/github-classroom-clone-assignments/784/1)
* [How to automatically gather or collect assignments?](https://education.github.community/t/how-to-automatically-gather-or-collect-assignments/2595)

[myrepos](https://myrepos.branchable.com) automates parallel management of a set of
repos. It doesn't create the initial repo set, which is that this tool does.

## License

MIT
