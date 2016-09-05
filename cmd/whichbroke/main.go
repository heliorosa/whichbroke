package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/heliorosa/whichbroke/vcs"
)

func showUsage(myself string) {
	fmt.Fprintf(os.Stderr, "usage: %s testCommand testArg1 testArg2 testArgn\n", myself)
}

func main() {
	// get current work dir
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't get current directory: %s\n", err.Error())
		os.Exit(1)
	}
	// we need at least one build command
	if len(os.Args) < 2 {
		showUsage(os.Args[0])
		os.Exit(2)
	}
	cmd := os.Args[1]
	// build arguments
	args := os.Args[2:]
	// open repo
	repo, err := vcs.OpenRepo(cwd, cmd, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "can't find a repo in the current directory")
		os.Exit(3)
	}
	// get commit log
	commits, err := repo.Log()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading the commit log: %s\n", err.Error())
		os.Exit(4)
	}
	// find first non passing commit to use as starting point
	var npIdx int
	for npIdx = 0; npIdx < len(commits); npIdx++ {
		ok, err := testCommit(repo, commits[npIdx])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error checking out the repo: %s\n", err.Error())
			os.Exit(5)
		}
		if !ok {
			break
		}
	}
	if npIdx == len(commits) {
		fmt.Fprintln(os.Stderr, "can't find a non passing commit/revision")
		os.Exit(6)
	}
	// find any passing commit before the non passing one
	var (
		pIdx int
		sz   = 1
	)
	for {
		pIdx = npIdx + sz
		if pIdx > len(commits)-1 {
			pIdx = len(commits) - 1
		}
		if pIdx == npIdx {
			fmt.Fprintf(os.Stderr, "can't find a passing commit/revision before %s\n", commits[npIdx])
			os.Exit(7)
		}
		ok, err := testCommit(repo, commits[pIdx])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error checking out the repo: %s\n", err.Error())
			os.Exit(5)
		}
		if ok {
			break
		}
		sz *= 2
	}
	// if pIdx==npIdx
	last := npIdx + sz/2
	sz = pIdx - last
	// bisect until the last passing commit is found
	for last != pIdx-1 {
		cIdx := last + sz/2
		ok, err := testCommit(repo, commits[cIdx])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error checking out the repo: %s\n", err.Error())
			os.Exit(5)
		}
		if ok {
			pIdx = cIdx
		} else {
			last = cIdx
		}
	}
	fmt.Fprintf(os.Stdout, "the last passing commit/revision is: %s\n", commits[pIdx])
}

// checkout a commit and try to run the tests
func testCommit(repo vcs.Repo, c string) (bool, error) {
	if err := repo.Checkout(c); err != nil {
		return false, err
	}
	if _, err := repo.TryBuild(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}
