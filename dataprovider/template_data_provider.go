package dataprovider

import (
	"log"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/cloudfoundry-incubator/golang-bump-progress/config"
	"github.com/cloudfoundry-incubator/golang-bump-progress/version"
)

const FETCH_INTERVAL = time.Minute

type Release struct {
	Name                        string
	URL                         string
	VersionOnDev                string
	ReleasedVersion             string
	FirstReleasedGolangVersion  string
	FirstReleasedReleaseVersion string
	BumpedInTas                 string
}

type TemplateData struct {
	Releases []Release
}

type versionFetcher interface {
	GetDevelopVersion(release config.Release) (string, error)
	GetReleasedVersion(release config.Release) (string, error)
	GetFirstReleasedVersion(release config.Release, releasedVersion string) (version.VersionInfo, error)
}

type tasVersionProvider interface {
	Fetch(ref string) error
	GetReleaseVersion(releaseName string) (string, bool)
}

type templateDataProvider struct {
	githubVersion versionFetcher
	tasVersion    tasVersionProvider
	config        config.Config
	lastFetchTime time.Time
	cachedData    TemplateData
}

func NewTemplateDataProvider(githubVersion versionFetcher, tasVersion tasVersionProvider, cfg config.Config) *templateDataProvider {
	return &templateDataProvider{
		githubVersion: githubVersion,
		tasVersion:    tasVersion,
		config:        cfg,
	}
}

func (p *templateDataProvider) Get() TemplateData {
	if p.lastFetchTime.IsZero() || p.lastFetchTime.Add(FETCH_INTERVAL).Before(time.Now()) {
		log.Println("fetching new data")
		p.lastFetchTime = time.Now()
		p.cachedData = p.fetch()
		return p.cachedData
	}

	return p.cachedData
}

func (p *templateDataProvider) fetch() TemplateData {
	data := TemplateData{
		Releases: []Release{},
	}
	err := p.tasVersion.Fetch("main")
	if err != nil {
		log.Printf("failed to get develop version for %s")
	}
	for _, release := range p.config.Releases {
		devVersion, err := p.githubVersion.GetDevelopVersion(release)
		if err != nil {
			log.Printf("failed to get develop version for %s: %s", release.Name, err.Error())
		}

		firstVersionInfo := version.VersionInfo{}
		releasedVersion, err := p.githubVersion.GetReleasedVersion(release)
		if err != nil {
			log.Printf("failed to get released version for %s: %s", release.Name, err.Error())
		} else {
			firstVersionInfo, err = p.githubVersion.GetFirstReleasedVersion(release, releasedVersion)
			if err != nil {
				log.Printf("failed to get first released minor version for %s: %s", release.Name, err.Error())
			}
		}

		bumpedInTas := p.bumpedInTas(release.Name, firstVersionInfo.ReleaseVersion)

		data.Releases = append(data.Releases, Release{
			Name:                        release.Name,
			URL:                         release.URL,
			VersionOnDev:                devVersion,
			ReleasedVersion:             releasedVersion,
			FirstReleasedGolangVersion:  firstVersionInfo.GolangVersion,
			FirstReleasedReleaseVersion: firstVersionInfo.ReleaseVersion,
			BumpedInTas:                 bumpedInTas,
		})
	}
	return data
}
func (p *templateDataProvider) bumpedInTas(releaseName string, firstReleaseVersion string) string {
	releaseTasVersion, found := p.tasVersion.GetReleaseVersion(releaseName)
	if !found {
		return "n/a"
	}

	if firstReleaseVersion == "" {
		return ""
	}
	firstReleaseV, err := semver.NewVersion(firstReleaseVersion)
	if err != nil {
		log.Printf("failed to parse release version for %s: %s", releaseName, err.Error())
		return ""
	}
	releaseTasV, err := semver.NewVersion(releaseTasVersion)
	if err != nil {
		log.Printf("failed to parse release version for %s: %s", releaseName, err.Error())
		return ""
	}
	if firstReleaseV.LessThan(releaseTasV) {
		return "yes"
	}

	return "no"
}
