package version

import (
	"context"

	"github.com/google/go-github/v54/github"
	"gopkg.in/yaml.v2"
)

const (
	TAS_RELEASES_FILE = "tas/Kilnfile.lock"
)

type KilnRelease struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type Kilnfile struct {
	Releases []KilnRelease `yaml:"releases"`
}

type tasVersion struct {
	githubClient *github.Client
	releases     map[string]string
	ctx          context.Context
}

func NewTasVersion(ctx context.Context, githubClient *github.Client) *tasVersion {
	return &tasVersion{
		githubClient: githubClient,
		ctx:          ctx,
	}
}

func (v *tasVersion) Fetch(ref string) error {
	kilnContents, _, _, err := v.githubClient.Repositories.GetContents(v.ctx, "pivotal", "tas", TAS_RELEASES_FILE, &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return err
	}
	kilnContent, err := kilnContents.GetContent()
	if err != nil {
		return err
	}

	var kilnFile Kilnfile
	err = yaml.Unmarshal([]byte(kilnContent), &kilnFile)
	if err != nil {
		return err
	}
	v.releases = map[string]string{}
	for _, release := range kilnFile.Releases {
		v.releases[release.Name] = release.Version
	}

	return nil
}

func (v *tasVersion) GetReleaseVersion(releaseName string) (string, bool) {
	version, ok := v.releases[releaseName]
	return version, ok

}
