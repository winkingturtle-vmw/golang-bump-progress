package version

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/google/go-github/v54/github"
)

const (
	GOLANG_BOSH_RELEASE_OWNER = "cloudfoundry"
	GOLANG_BOSH_RELEASE_REPO  = "bosh-package-golang-release"
	GOLANG_LINUX_PACKAGE      = "golang-1-linux"
	GOLANG_WINDOWS_PACKAGE    = "golang-1-windows"
	WARMUP_COMMITS            = 20
	FILES_IN_FINAL_RELEASE    = 20
)

var (
	FINGERPRINT_PATCH_RE = regexp.MustCompile(`.*\+\s+version:\s(\w+).+`)
	BLOB_URL_RE          = regexp.MustCompile(`.+/blob/(\w+)/.+`)
)

type PackageSpec struct {
	Name        string `yaml:"name"`
	Fingerprint string `yaml:"fingerprint"`
}

type boshPackageVersion struct {
	githubClient         *github.Client
	fingerprintsCache    map[string]string
	fingerprintsCacheMux sync.Mutex
	ctx                  context.Context
}

func NewBoshPackageVersion(ctx context.Context, githubClient *github.Client) *boshPackageVersion {
	return &boshPackageVersion{
		fingerprintsCache:    map[string]string{},
		fingerprintsCacheMux: sync.Mutex{},
		githubClient:         githubClient,
		ctx:                  ctx,
	}
}

func (v *boshPackageVersion) PopulateCache() error {
	v.fingerprintsCacheMux.Lock()
	defer v.fingerprintsCacheMux.Unlock()
	linuxVersionFile := fmt.Sprintf("packages/%s/version", GOLANG_LINUX_PACKAGE)
	windowsVersionFile := fmt.Sprintf("packages/%s/version", GOLANG_WINDOWS_PACKAGE)
	linuxFingerprintFile := fmt.Sprintf(`.final_builds/packages/%s/index.yml`, GOLANG_LINUX_PACKAGE)
	windowsFingerprintFile := fmt.Sprintf(`.final_builds/packages/%s/index.yml`, GOLANG_WINDOWS_PACKAGE)

	// only pulling commits for Linux, since commits to Linux final.yml also update Windows final.yml
	commitResults, _, err := v.githubClient.Repositories.ListCommits(
		v.ctx,
		GOLANG_BOSH_RELEASE_OWNER,
		GOLANG_BOSH_RELEASE_REPO,
		&github.CommitsListOptions{
			Path:        linuxFingerprintFile,
			ListOptions: github.ListOptions{PerPage: WARMUP_COMMITS},
		})
	if err != nil {
		return err
	}

	for _, commitResult := range commitResults {
		commitSHA := commitResult.GetSHA()
		fmt.Printf("analyzing commit %s\n", commitSHA)
		commit, _, err := v.githubClient.Repositories.GetCommit(
			v.ctx,
			GOLANG_BOSH_RELEASE_OWNER,
			GOLANG_BOSH_RELEASE_REPO,
			commitSHA,
			&github.ListOptions{PerPage: FILES_IN_FINAL_RELEASE},
		)
		if err != nil {
			return err
		}

		if len(commit.Files) < 1 {
			return fmt.Errorf("failed to get files for %s", commitSHA)
		}

		for _, file := range commit.Files {
			switch file.GetFilename() {
			case linuxFingerprintFile:
				fingerprint, version, err := v.getFingerprintVersionFromPatch(file.GetPatch(), linuxVersionFile, commitSHA)
				fmt.Printf("linux fingerprint: %#v\n", fingerprint)
				if err != nil {
					return err
				}
				v.fingerprintsCache[fingerprint] = version
			case windowsFingerprintFile:
				fingerprint, version, err := v.getFingerprintVersionFromPatch(file.GetPatch(), windowsVersionFile, commitSHA)
				fmt.Printf("windows fingerprint: %#v\n", fingerprint)
				if err != nil {
					return err
				}
				v.fingerprintsCache[fingerprint] = version
			}

		}
	}
	fmt.Printf("fingerprintsCache: %#v\n", v.fingerprintsCache)
	return nil
}

func (v *boshPackageVersion) GetFingerprintVersion(fingerprint string, golangPackage string) (string, error) {
	v.fingerprintsCacheMux.Lock()
	defer v.fingerprintsCacheMux.Unlock()
	if version, ok := v.fingerprintsCache[fingerprint]; ok {
		fmt.Println("using cache")
		return version, nil
	}
	fmt.Println("not using cache")

	query := fmt.Sprintf("%s repo:%s/%s", fingerprint, GOLANG_BOSH_RELEASE_OWNER, GOLANG_BOSH_RELEASE_REPO)

	searchResult, _, err := v.githubClient.Search.Code(v.ctx, query, nil)
	if err != nil {
		return "", err
	}
	if searchResult.GetTotal() < 1 {
		return "", errors.New("no results for bosh package fingerprint")
	}

	codeResult := searchResult.CodeResults[0]
	refURL := codeResult.GetHTMLURL()
	matches := BLOB_URL_RE.FindSubmatch([]byte(refURL))
	if len(matches) < 2 {
		return "", errors.New("failed to parse ref")
	}

	golangVersionRef := string(matches[1])

	versionFile := fmt.Sprintf("packages/%s/version", golangPackage)
	version, err := v.getFileContentsForRef(versionFile, golangVersionRef)
	if err != nil {
		return "", err
	}

	v.fingerprintsCache[fingerprint] = version
	return version, nil
}

func (v *boshPackageVersion) getFingerprintVersionFromPatch(patch string, versionFile string, commitSHA string) (string, string, error) {
	matches := FINGERPRINT_PATCH_RE.FindSubmatch([]byte(patch))
	if len(matches) < 2 {
		return "", "", fmt.Errorf("failed to parse patch for sha %s", commitSHA)
	}

	fingerprint := string(matches[1])
	version, err := v.getFileContentsForRef(versionFile, commitSHA)
	if err != nil {
		return "", "", err
	}
	return fingerprint, version, nil
}

func (v *boshPackageVersion) getFileContentsForRef(filePath string, ref string) (string, error) {
	versionContent, _, _, err := v.githubClient.Repositories.GetContents(
		v.ctx,
		GOLANG_BOSH_RELEASE_OWNER,
		GOLANG_BOSH_RELEASE_REPO,
		filePath,
		&github.RepositoryContentGetOptions{Ref: ref},
	)
	if err != nil {
		return "", err
	}
	content, err := versionContent.GetContent()
	if err != nil {
		return "", err
	}
	return content, nil
}
