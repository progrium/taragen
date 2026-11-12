package design

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Git answers the questions the workflow asks of git.
type Git interface {
	// BranchTime returns the time of the newest commit on branch, looking at
	// both refs/remotes/origin/<branch> and refs/heads/<branch>. ok is false
	// if neither ref exists.
	BranchTime(branch string) (t time.Time, ok bool)
}

func isGitRoot(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".git"))
	return err == nil
}

type execGit struct{ dir string }

func (g execGit) BranchTime(branch string) (time.Time, bool) {
	var newest time.Time
	found := false
	for _, ref := range []string{"refs/remotes/origin/" + branch, "refs/heads/" + branch} {
		out, err := exec.Command("git", "-C", g.dir, "log", "-1", "--format=%ct", ref, "--").Output()
		if err != nil {
			continue
		}
		sec, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		if err != nil {
			continue
		}
		t := time.Unix(sec, 0)
		if !found || t.After(newest) {
			newest = t
		}
		found = true
	}
	return newest, found
}

// StaticGit is a Git backed by a fixed table, for tests and fixtures.
type StaticGit map[string]time.Time

func (g StaticGit) BranchTime(branch string) (time.Time, bool) {
	t, ok := g[branch]
	return t, ok
}

// TrackedFiles lists the files check reads, relative to root. In a git
// checkout these are the tracked and untracked-but-not-ignored files;
// elsewhere (fixtures) every file.
func TrackedFiles(root string) ([]string, error) {
	if isGitRoot(root) {
		out, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached", "--others", "--exclude-standard").Output()
		if err != nil {
			return nil, err
		}
		var files []string
		for _, f := range strings.Split(string(out), "\x00") {
			if f != "" {
				files = append(files, f)
			}
		}
		return files, nil
	}
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	return files, err
}
