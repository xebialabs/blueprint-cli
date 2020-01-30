package xl

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/xebialabs/xl-cli/pkg/util"
	"gopkg.in/src-d/go-git.v4"
)

//////// Generic interface /////////////////////////////////////////////////////////////////////////////////////////////

type Repo interface {
	SCM() string
	Remote() (string, error)
	LocalPath() string
	IsDirty() (bool, error)
	LatestCommitInfo() (*CommitInfo, error)
}

type CommitInfo struct {
	Commit  string
	Author  string
	Date    time.Time
	Message string
}

func NewLocalRepo(location string) (Repo, error) {
	// here we can detect and switch between implementations
	// now only support for git
	return NewGitRepo(location)
}

func FindRepo(file string) (Repo, error) {
	base := file
	for {
		parent := filepath.Dir(base)
		base = parent
		util.Verbose("Trying to find SCM repo in folder: %s \n", parent)
		repo, err := NewLocalRepo(parent)
		if err == nil {
			return repo, nil
		} else if base == filepath.Dir(base) {
			// break because we are at the root
			return nil, fmt.Errorf("cannot determine SCM for: %s", file)
		}
	}
	return nil, fmt.Errorf("cannot determine SCM for: %s", file)
}

//////// Git implementation ////////////////////////////////////////////////////////////////////////////////////////////

type GitRepo struct {
	location   string
	Repository *git.Repository
}

func NewGitRepo(location string) (*GitRepo, error) {
	r, err := git.PlainOpen(location)
	if err == nil {
		return &GitRepo{location, r}, nil
	} else {
		return nil, err
	}
}

func (s GitRepo) SCM() string {
	return "git"
}

func (s GitRepo) Remote() (string, error) {
	remote, err := s.Repository.Remote("origin")
	if err != nil {
		return "", err
	}
	if len(remote.Config().URLs) == 0 {
		return "", fmt.Errorf("error while trying to get remote url for origin: URL not set")
	}
	return remote.Config().URLs[0], nil
}

func (s GitRepo) LocalPath() string {
	return s.location
}

func (s GitRepo) IsDirty() (bool, error) {
	worktree, err := s.Repository.Worktree()
	if err != nil {
		return false, err
	}
	statuses, err := worktree.Status()
	if err != nil {
		return false, err
	}
	return !statuses.IsClean(), nil
}

func (s GitRepo) LatestCommitInfo() (*CommitInfo, error) {
	head, err := s.Repository.Head()
	if err != nil {
		return nil, err
	}
	commit, err := s.Repository.CommitObject(head.Hash())
	if err != nil {
		return nil, err
	}
	return &CommitInfo{commit.Hash.String(), fmt.Sprintf("%s <%s>", commit.Author.Name, commit.Author.Email), commit.Author.When, commit.Message}, nil
}
