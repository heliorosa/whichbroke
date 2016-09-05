package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

var (
	gitRe, hgRe, bzrRe *regexp.Regexp
)

func init() {
	var err error
	if gitRe, err = regexp.Compile("(?m)^commit ([[:xdigit:]]{40})$"); err != nil {
		panic(err)
	}
	if hgRe, err = regexp.Compile("(?m)^changeset:\\w*(.*:.*)$"); err != nil {
		panic(err)
	}
	if bzrRe, err = regexp.Compile("(?m)^revno: (.*)$"); err != nil {
		panic(err)
	}
}

const startLogSize = 256

// Repo is simple and generic inteface for a git repository
type Repo interface {
	Log() ([]string, error)
	Checkout(commit string) error
	TryBuild() ([]byte, error)
}

// GitRepo implements Repo
type GitRepo struct{ *baseRepo }

func newGitRepo(path, buildCmd string, buildArgs []string) *GitRepo {
	return &GitRepo{&baseRepo{
		path:         path,
		vcsCmd:       "git",
		logArgs:      []string{"log"},
		commitRegExp: gitRe,
		chechoutArgs: []string{"checkout"},
		buildCmd:     buildCmd,
		buildArgs:    buildArgs,
	}}
}

// HgRepo implements Repo
type HgRepo struct{ *baseRepo }

func newHgRepo(path, buildCmd string, buildArgs []string) *HgRepo {
	return &HgRepo{&baseRepo{
		path:         path,
		vcsCmd:       "hg",
		logArgs:      []string{"log"},
		commitRegExp: hgRe,
		chechoutArgs: []string{"revert", "--all", "-r"},
		buildCmd:     buildCmd,
		buildArgs:    buildArgs,
	}}
}

// BzrRepo implements Repo
type BzrRepo struct{ *baseRepo }

func newBzrRepo(path, buildCmd string, buildArgs []string) *BzrRepo {
	return &BzrRepo{&baseRepo{
		path:         path,
		vcsCmd:       "bzr",
		logArgs:      []string{"log"},
		commitRegExp: bzrRe,
		chechoutArgs: []string{"revert", "-r"},
		buildCmd:     buildCmd,
		buildArgs:    buildArgs,
	}}
}

type baseRepo struct {
	path         string         // repo path
	vcsCmd       string         // vcs command
	logArgs      []string       // arguments for log
	commitRegExp *regexp.Regexp // reg exp to extract commits/revisions
	chechoutArgs []string       // arguments for checkout
	buildCmd     string         // build command
	buildArgs    []string       // build args
}

func (r *baseRepo) Log() ([]string, error) {
	c := runAtPath(r.path, r.vcsCmd, r.logArgs)
	out, err := c.Output()
	if err != nil {
		return nil, err
	}
	retb := r.commitRegExp.FindAllSubmatch(out, -1)
	commits := make([]string, 0, len(retb))
	for _, a := range retb {
		commits = append(commits, string(a[1]))
	}
	return commits, nil
}

func (r *baseRepo) Checkout(commit string) error {
	args := make([]string, len(r.chechoutArgs)+1)
	copy(args, r.chechoutArgs)
	args[len(args)-1] = commit
	c := runAtPath(r.path, r.vcsCmd, args)
	_, err := c.Output()
	return err
}

func (r *baseRepo) TryBuild() ([]byte, error) {
	c := runAtPath(r.path, r.buildCmd, r.buildArgs)
	return c.Output()
}

func runAtPath(path, cmd string, args []string) *exec.Cmd {
	c := exec.Command(cmd, args...)
	c.Dir = path
	return c
}

type RepoError struct{ Path string }

func (e *RepoError) Error() string { return "no repository found: " + e.Path }

// OpenRepo checks if a given directory contains a repository
// and returns a Repo and nil as error if succeeds, otherwise
// returns an nil and and error of type *RepoError
func OpenRepo(path, buildCmd string, buildArgs []string) (Repo, error) {
	if hasSubDir(path, ".git") {
		return newGitRepo(path, buildCmd, buildArgs), nil
	}
	if hasSubDir(path, ".hg") {
		return newHgRepo(path, buildCmd, buildArgs), nil
	}
	if hasSubDir(path, ".bzr") {
		return newBzrRepo(path, buildCmd, buildArgs), nil
	}
	return nil, &RepoError{path}
}

func hasSubDir(path, sub string) bool {
	fi, err := os.Stat(filepath.Join(path, sub))
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			return false
		}
		panic(err)
	}
	return fi.IsDir()
}

var _ Repo = (*GitRepo)(nil)
var _ Repo = (*HgRepo)(nil)
var _ Repo = (*BzrRepo)(nil)
