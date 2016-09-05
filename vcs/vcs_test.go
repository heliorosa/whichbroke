package vcs

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func unpackTestRepo(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tarball := tar.NewReader(gz)
	tmpDir, err := ioutil.TempDir("", "test_data")
	if err != nil {
		return "", err
	}
	for {
		var h *tar.Header
		if h, err = tarball.Next(); err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("can't find next header: %s", err.Error())
		}
		if hinfo := h.FileInfo(); hinfo.IsDir() {
			if err := os.MkdirAll(filepath.Join(tmpDir, h.Name), hinfo.Mode()); err != nil {
				return "", fmt.Errorf("can't mkdir(): %s", err.Error())
			}
		} else {
			f, err := os.OpenFile(filepath.Join(tmpDir, h.Name), os.O_RDWR|os.O_CREATE|os.O_TRUNC, hinfo.Mode())
			if err != nil {
				return "", fmt.Errorf("can't open file: %s", err.Error())
			}
			defer f.Close()
			if n, err := io.Copy(f, tarball); err != nil {
				return "", fmt.Errorf("can't write file: %s", err.Error())
			} else if n != h.Size {
				return "", fmt.Errorf("file %s: only %d of %d bytes written", h.Name, n, hinfo.Size())
			}
			f.Close()
		}
	}
	return tmpDir, nil
}

func testRepo(t *testing.T, repo Repo) {
	// get log
	log, err := repo.Log()
	if err != nil {
		t.Error(err)
		return
	}
	// try to build the last commit (must fail)
	_, err = repo.TryBuild()
	if err == nil {
		t.Error("should not build")
		return
	}
	// checkout the first commit
	if err = repo.Checkout(log[len(log)-1]); err != nil {
		t.Error(err)
		return
	}
	// try to build the first commit (must succeed)
	_, err = repo.TryBuild()
	if err != nil {
		t.Error(err)
		return
	}
}

func testData(t string) string { return "./testdata/" + t + ".tgz" }

func TestGitRepo(t *testing.T) {
	// unpack test
	tmpDir, err := unpackTestRepo(testData("git"))
	if err != nil {
		t.Error(err)
		return
	}
	// create repo
	testRepo(t, newGitRepo(tmpDir, "go", []string{"test", "-v"}))
}

func TestHgRepo(t *testing.T) {
	// unpack test
	tmpDir, err := unpackTestRepo(testData("hg"))
	if err != nil {
		t.Error(err)
		return
	}
	// test repo
	testRepo(t, newHgRepo(tmpDir, "go", []string{"test", "-v"}))
}

func TestBzrRepo(t *testing.T) {
	// unpack test
	tmpDir, err := unpackTestRepo(testData("bzr"))
	if err != nil {
		t.Error(err)
		return
	}
	// test repo
	testRepo(t, newBzrRepo(tmpDir, "go", []string{"test", "-v"}))
}

func TestOpenRepo(t *testing.T) {
	// unpack test repos
	tmpDirGit, err := unpackTestRepo(testData("git"))
	if err != nil {
		t.Error(err)
		return
	}
	tmpDirHg, err := unpackTestRepo(testData("hg"))
	if err != nil {
		t.Error(err)
		return
	}
	tmpDirBzr, err := unpackTestRepo(testData("bzr"))
	if err != nil {
		t.Error(err)
		return
	}
	cmd := "go"
	args := []string{"test", "-v"}
	repo, err := OpenRepo(tmpDirGit, cmd, args)
	if err != nil {
		t.Error(err)
		return
	}
	if _, ok := repo.(*GitRepo); !ok {
		t.Error("expected a git repo")
		return
	}
	if repo, err = OpenRepo(tmpDirHg, cmd, args); err != nil {
		t.Error(err)
		return
	}
	if _, ok := repo.(*HgRepo); !ok {
		t.Error("expected a hg repo")
		return
	}
	if repo, err = OpenRepo(tmpDirBzr, cmd, args); err != nil {
		t.Error(err)
		return
	}
	if _, ok := repo.(*BzrRepo); !ok {
		t.Error("expected a bzr repo")
		return
	}
}
