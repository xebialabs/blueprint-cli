package zip

import (
    "archive/zip"
    "fmt"
    "github.com/xebialabs/blueprint-cli/pkg/blueprint/repository"
    "github.com/xebialabs/blueprint-cli/pkg/blueprint/repository/local"
    "github.com/xebialabs/blueprint-cli/pkg/models"
    "github.com/xebialabs/blueprint-cli/pkg/util"
    "io"
    "net/http"
    "os"
    "os/user"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
)

const (
    ZipRepo string = "ziprepo"
)

// Local Blueprint Repository Provider implementation
type ZipBlueprintRepository struct {
    Name               string
    Path               string
    delegateRepository repository.BlueprintRepository
    confMap            map[string]string
}

func newZipBlueprintRepository(confMap map[string]string) (*ZipBlueprintRepository, error) {
    repo := new(ZipBlueprintRepository)
    repo.Name = confMap["name"]
    repo.confMap = confMap

    // expand home dir if needed
    if !util.MapContainsKeyWithVal(confMap, "path") {
        return nil, fmt.Errorf("'path' config field must be set for Zip repository type")
    }

    repo.Path = confMap["path"]
    if err := os.MkdirAll(ZipRepo, 0755); err != nil {
        return nil, fmt.Errorf("cannot make local direactory %s: %s", ZipRepo, err.Error())
    }
    return repo, nil
}

func NewDefaultZipBlueprintRepository(confMap map[string]string, CLIVersion string) (*ZipBlueprintRepository, error) {
    if repo, err := newZipBlueprintRepository(confMap); err == nil {
        currentUser, err := user.Current()
        if err != nil {
            return nil, fmt.Errorf("cannot get current user: %s", err.Error())
        }

        localPath := readHttpZipContent(repo.Path, CLIVersion)
        zipDir := getCLIVersionLocation(util.ExpandHomeDirIfNeeded(localPath, currentUser), CLIVersion)
        if err = unzip(zipDir, ZipRepo); err != nil {
            return nil, fmt.Errorf("cannot unzip file %s: %s", zipDir, err.Error())
        }

        return repo, nil
    } else {
        return nil, err
    }
}

func NewCustomZipBlueprintRepository(confMap map[string]string) (*ZipBlueprintRepository, error) {
    if repo, err := newZipBlueprintRepository(confMap); err == nil {
        currentUser, err := user.Current()
        if err != nil {
            return nil, fmt.Errorf("cannot get current user: %s", err.Error())
        }

        localPath := readDirectHttpZipContent(repo.Path)
        zipDir := util.ExpandHomeDirIfNeeded(localPath, currentUser)
        if err = unzip(zipDir, ZipRepo); err != nil {
            return nil, fmt.Errorf("cannot unzip file %s: %s", zipDir, err.Error())
        }

        return repo, nil
    } else {
        return nil, err
    }
}

func (repo *ZipBlueprintRepository) Initialize() error {
    localRepoConfMap := make(map[string]string)
    for key, value := range repo.confMap {
        localRepoConfMap[key] = value
    }
    localRepoConfMap["name"] = "Delegate - " + localRepoConfMap["name"]
    localRepoConfMap["path"] = ZipRepo

    localRepository, err := local.NewLocalBlueprintRepository(localRepoConfMap)
    repo.delegateRepository = localRepository
    return err
}

func (repo *ZipBlueprintRepository) GetName() string {
    return repo.Name
}

func (repo *ZipBlueprintRepository) GetProvider() string {
    return models.ProviderZip
}

func (repo *ZipBlueprintRepository) GetInfo() string {
    if delegateRepoValue, ok := repo.delegateRepository.(*local.LocalBlueprintRepository); ok {
        return fmt.Sprintf(
            "Provider: %s\n  Repository name: %s\n Path: %s\n Local path: %s\n  Ignored directories: %s\n  Ignored files: %s",
            repo.GetProvider(),
            repo.Name,
            repo.Path,
            ZipRepo,
            delegateRepoValue.IgnoredDirs,
            delegateRepoValue.IgnoredFiles,
        )
    } else {
        util.Fatal("Cannot cast delegate repository to local repository")
        return ""
    }
}

func (repo *ZipBlueprintRepository) ListBlueprintsFromRepo() (map[string]*models.BlueprintRemote, []string, error) {
    return repo.delegateRepository.ListBlueprintsFromRepo()
}

func (repo *ZipBlueprintRepository) GetFileContents(filePath string) (*[]byte, error) {
    return repo.delegateRepository.GetFileContents(filePath)
}

func unzip(src, dest string) error {
    dest = filepath.Clean(dest) + string(os.PathSeparator)

    util.Verbose("Unzipping file %s to the destination %s\n", src, dest)

    r, err := zip.OpenReader(src)
    if err != nil {
        return err
    }
    defer func() {
        if err := r.Close(); err != nil {
            panic(err)
        }
    }()

    // Closure to address file descriptors issue with all the deferred .Close() methods
    extractAndWriteFile := func(f *zip.File) error {
        path := filepath.Join(dest, f.Name)
        // Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
        if !strings.HasPrefix(path, dest) {
            return fmt.Errorf("%s: illegal file path", path)
        }

        rc, err := f.Open()
        if err != nil {
            return err
        }
        defer func() {
            if err := rc.Close(); err != nil {
                panic(err)
            }
        }()

        if f.FileInfo().IsDir() {
            os.MkdirAll(path, f.Mode())
        } else {
            os.MkdirAll(filepath.Dir(path), f.Mode())
            f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
            if err != nil {
                return err
            }
            defer func() {
                if err := f.Close(); err != nil {
                    panic(err)
                }
            }()

            _, err = io.Copy(f, rc)
            if err != nil {
                return err
            }
        }
        return nil
    }

    for _, f := range r.File {
        err := extractAndWriteFile(f)
        if err != nil {
            return err
        }
    }

    return nil
}

func readHttpZipContent(zipLocation string, CLIVersion string) string {
    if strings.Index(zipLocation, "http") == 0 {
        var zipPath = getCLIVersionLocation(zipLocation, CLIVersion)
        return readDirectHttpZipContent(zipPath)
    }
    return zipLocation
}

func readDirectHttpZipContent(zipLocation string) string {
    if strings.Index(zipLocation, "http") == 0 {
        util.Verbose("Downloading operator zip %s to %s\n", zipLocation, ZipRepo)
        client := http.Client{
            CheckRedirect: func(r *http.Request, via []*http.Request) error {
                r.URL.Opaque = r.URL.Path
                return nil
            },
        }
        resp, err := client.Get(zipLocation)
        if err != nil {
            util.Fatal("Cannot download operator zip file [%s]: %s\n", zipLocation, err)
        }
        defer resp.Body.Close()
        if resp.StatusCode == http.StatusOK {
            zipPathSegments := strings.Split(resp.Request.URL.Path, "/")
            zipPath := filepath.FromSlash(ZipRepo + "/" + zipPathSegments[len(zipPathSegments)-1])
            out, err := os.Create(zipPath)
            if err != nil {
                util.Fatal("Cannot create operator zip file [%s] on filesystem: %s\n", zipPath, err)
            }
            defer out.Close()

            if size, err := io.Copy(out, resp.Body); err != nil {
                util.Fatal("Cannot save operator zip file [%s] on filesystem: %s\n", zipPath, err)
            } else {
                util.Verbose("Downloaded a operator zip %s with size %d\n", zipPath, size)
            }
            return zipPath
        } else {
            util.Fatal("Cannot download operator zip file [%s]: status response %s\n", zipLocation, resp.Status)
        }
    }
    return zipLocation
}

func getCLIVersionLocation(location, CLIVersion string) string {
    if strings.Contains(location, models.BlueprintCurrentCLIVersion) {
        evaluatedLocation := strings.Replace(location, models.BlueprintCurrentCLIVersion, CLIVersion, -1)
        if locationExists(evaluatedLocation) {
            return evaluatedLocation
        }
        re := regexp.MustCompile("^([0-9]+).([0-9]+).([0-9]+)")
        versions := re.FindStringSubmatch(CLIVersion)
        if len(versions) == 4 {
            // Tick down like a reverse odometer until an existing blueprint directory is found
            yDigit, _ := strconv.Atoi(versions[2])
            for yDigit >= 0 {
                zDigit, _ := strconv.Atoi(versions[3])
                for zDigit >= 0 {
                    // Match on x.y.z
                    evaluatedLocation := strings.Replace(location, models.BlueprintCurrentCLIVersion, fmt.Sprintf("%s.%d.%d", versions[1], yDigit, zDigit), -1)
                    if locationExists(evaluatedLocation) {
                        return evaluatedLocation
                    }
                    zDigit--
                }
                // Match on x.y
                evaluatedLocation := strings.Replace(location, models.BlueprintCurrentCLIVersion, fmt.Sprintf("%s.%d", versions[1], yDigit), -1)
                if locationExists(evaluatedLocation) {
                    return evaluatedLocation
                }
                yDigit--
            }
        }
    }
    return location
}

func locationExists(location string) bool {
    if strings.Index(location, "http") == 0 {
        return util.URLExists(location)
    } else {
        return util.FileExists(location)
    }
}
