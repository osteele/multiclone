# Multiclone

Clone all the forks of a repository, or all the repos of a [GitHub Classroom](https://classroom.github.com) assignment.

This is useful for collecting and reviewing assignments and student projects.

Features:

* Written in Golang for easier distribution. (I got tired juggling of juggling Anaconda / virtualenv between various classroom and tool environments.)
* Automatic repo discovery. Knows about “students fork” and “GitHub Classroom” conventions.
* Repos are cloned in parallel.
* Create a [myrepos](https://myrepos.branchable.com) `.mrconfig` file.

## Usage

    multiclone https://github.com/owner/repo [DIR]
    multiclone owner/repo [DIR]

Clone forks of owner/repo into DIR (or the current directory).

### GitHub Classroom

    multiclone https://github.com/owner/repo [DIR] --classroom
    multiclone org/repo [DIR] --classroom

Clone org's repos named repo-* into DIR (or the current directory).

This is intended for use with repos created via [GitHub Classroom](https://classroom.github.com).

### Options

    multiclone owner/repo [DIR] --dry-run

See the `git` commands that would be run, without actually running them.

    multiclone --help

Lists additional options.

## Install

1. **Install go** (1) via [Homebrew](https://brew.sh): `brew install go`; or (2) [download](https://golang.org/doc/install#tarball).
2. `go install github.com/osteele/multiclone`
3. Create a [GitHub personal access token for the command line](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/)
4. Set `GITHUB_TOKEN` to this value: `export GITHUB_TOKEN=…`

## Alternatives

These [GitHub Education Community](https://education.github.community/t/how-to-automatically-gather-or-collect-assignments/2595) forum threads discuss a variety of alternatives (including one I wrote before I wrote this):

* [GitHub Classroom: clone assignments](https://education.github.community/t/github-classroom-clone-assignments/784/1)
* [How to automatically gather or collect assignments?](https://education.github.community/t/how-to-automatically-gather-or-collect-assignments/2595)

[myrepos](https://myrepos.branchable.com) automates parallel management of a set of
repos. It doesn't create the initial repo set, which is that this tool does.

## License

MIT
