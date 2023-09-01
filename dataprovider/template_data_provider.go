package dataprovider

import (
	"log"
	"time"

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
}

type TemplateData struct {
	Releases []Release
}

type versionFetcher interface {
	GetDevelopVersion(release config.Release) (string, error)
	GetReleasedVersion(release config.Release) (string, error)
	GetFirstReleasedVersion(release config.Release, releasedVersion string) (version.VersionInfo, error)
}

type templateDataProvider struct {
	githubVersion versionFetcher
	config        config.Config
	lastFetchTime time.Time
	cachedData    TemplateData
}

func NewTemplateDataProvider(githubVersion versionFetcher, cfg config.Config) *templateDataProvider {
	return &templateDataProvider{
		githubVersion: githubVersion,
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
	for _, release := range p.config.Releases {
		devVersion, err := p.githubVersion.GetDevelopVersion(release)
		if err != nil {
			log.Printf("failed to get develop version for %s: %s", release.Name, err.Error())
		}

		versionInfo := version.VersionInfo{}
		releasedVersion, err := p.githubVersion.GetReleasedVersion(release)
		if err != nil {
			log.Printf("failed to get released version for %s: %s", release.Name, err.Error())
		} else {
			versionInfo, err = p.githubVersion.GetFirstReleasedVersion(release, releasedVersion)
			if err != nil {
				log.Printf("failed to get first released minor version for %s: %s", release.Name, err.Error())
			}
		}

		data.Releases = append(data.Releases, Release{
			Name:                        release.Name,
			URL:                         release.URL,
			VersionOnDev:                devVersion,
			ReleasedVersion:             releasedVersion,
			FirstReleasedGolangVersion:  versionInfo.GolangVersion,
			FirstReleasedReleaseVersion: versionInfo.ReleaseVersion,
		})
	}
	return data
}
