package xl

import (
    "testing"
    "gopkg.in/src-d/go-git.v4"
    "io/ioutil"
    "github.com/stretchr/testify/assert"
    "fmt"
    "path/filepath"
    "gopkg.in/src-d/go-git.v4/plumbing/object"
    "time"
    "os"
    "gopkg.in/src-d/go-git.v4/config"
)

var testDate = time.Date(2019, time.April, 17, 0, 0, 0, 0, time.UTC)
var testUsername = "John Doe"
var testEmail = "john@doe.org"
var testMessage = "test commit message"

func TestVCS(t *testing.T) {

    t.Run("should get basic repo and commit info", func(t *testing.T) {

        gitRepository, artifactsDir := initDefaultGitRepo(t)
        defer os.RemoveAll(artifactsDir)

        repo, err := NewLocalRepo(artifactsDir)
        checkError(t, err)

        assert.Equal(t, "git", repo.Vcs())
        assert.Equal(t, artifactsDir, repo.LocalPath())

        dirty, err := repo.IsDirty()
        checkError(t, err)
        assert.False(t, dirty)

        info, err := repo.LatestCommitInfo()
        checkError(t, err)

        head, err := gitRepository.Head()
        checkError(t, err)

        assert.Equal(t, head.Hash().String(), info.Commit)
        assert.Equal(t, testMessage, info.Message)
        assert.Equal(t, fmt.Sprintf("%s <%s>", testUsername, testEmail), info.Author)
        assert.Equal(t, testDate.Unix(), info.Date.Unix())

    })

    t.Run("should detect repo from subfolder", func(t *testing.T) {
        gitRepository, artifactsDir := initDefaultGitRepo(t)
        defer os.RemoveAll(artifactsDir)

        repo, err := FindRepo(filepath.Join(artifactsDir, "subfolder", "xl.yaml"))
        checkError(t, err)

        info, err := repo.LatestCommitInfo()
        checkError(t, err)

        head, err := gitRepository.Head()
        checkError(t, err)

        assert.Equal(t, head.Hash().String(), info.Commit)
    })

    t.Run("should return error when can not detect repo", func(t *testing.T) {
        artifactsDir, err := ioutil.TempDir("", "gitrepo")
        checkError(t, err)
        defer os.RemoveAll(artifactsDir)

        repo, err := FindRepo(filepath.Join(artifactsDir))

        assert.Equal(t, err.Error(), fmt.Sprintf("cannot determine VCS for folder: %s", artifactsDir))
        assert.Nil(t, repo)
    })

    t.Run("should detect remote origin from repo", func(t *testing.T) {
        gitRepository, artifactsDir := initDefaultGitRepo(t)
        defer os.RemoveAll(artifactsDir)

        config := &config.RemoteConfig{Name:"origin", URLs:[]string{"http://github.com/xebialabs/devops-as-code"}}
        gitRepository.CreateRemote(config)

        repo, err := NewLocalRepo(artifactsDir)
        checkError(t, err)

        remote, err := repo.Remote()
        checkError(t, err)

        assert.Equal(t, "http://github.com/xebialabs/devops-as-code", remote)
    })

    t.Run("should indicate dirty when there are untracked files", func(t *testing.T) {
        _, artifactsDir := initDefaultGitRepo(t)
        defer os.RemoveAll(artifactsDir)

        filename := filepath.Join(artifactsDir, "new.yaml")
        err := ioutil.WriteFile(filename, []byte(""), 0644)
        checkError(t, err)

        repo, err := NewLocalRepo(artifactsDir)
        checkError(t, err)

        dirty, err := repo.IsDirty()
        checkError(t, err)
        assert.True(t, dirty)
    })

}

func initDefaultGitRepo(t *testing.T) (*git.Repository, string){
    artifactsDir, err := ioutil.TempDir("", "gitrepo")
    checkError(t, err)

    repository, err := git.PlainInit(artifactsDir, false)
    checkError(t, err)

    w, err := repository.Worktree()
    checkError(t, err)

    filename := filepath.Join(artifactsDir, "xl.yaml")
    err2 := ioutil.WriteFile(filename, []byte(""), 0644)
    checkError(t, err2)

    _, err = w.Add("xl.yaml")
    checkError(t, err)

    folder1 := filepath.Join(artifactsDir, "subfolder")
    os.Mkdir(folder1, 0755)

    filename2 := filepath.Join(folder1, "xl.yaml")
    err3 := ioutil.WriteFile(filename2, []byte(""), 0644)
    checkError(t, err3)

    _, err = w.Add("subfolder/xl.yaml")
    checkError(t, err)

    commit, err := w.Commit(testMessage, &git.CommitOptions{
        Author: &object.Signature{
            Name:  testUsername,
            Email: testEmail,
            When:  testDate,
        },
    })
    checkError(t, err)

    obj, err := repository.CommitObject(commit)
    checkError(t, err)

    fmt.Println(obj)
    return repository, artifactsDir
}

func checkError(t *testing.T, err error) {
    if err != nil {
        assert.FailNow(t, "Error: %s", err)
    }

}
