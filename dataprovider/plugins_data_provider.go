package dataprovider

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/cloudfoundry-incubator/golang-bump-progress/config"
	"github.com/google/go-github/v54/github"
)

type Plugin struct {
	Name            string
	URL             string
	DevVersion      string
	ReleasedVersion string
	AllBumped       bool
}

type PluginsData struct {
	Plugins []Plugin
}

type pluginsDataProvider struct {
	config        config.Config
	lastFetchTime time.Time
	cachedData    PluginsData
	githubClient  *github.Client
	ctx           context.Context
}

func NewPluginsDataProvider(ctx context.Context, githubClient *github.Client, cfg config.Config) *pluginsDataProvider {
	return &pluginsDataProvider{
		config:       cfg,
		githubClient: githubClient,
		ctx:          ctx,
	}
}

func (p *pluginsDataProvider) Get(targetGoVersion string) PluginsData {
	if p.lastFetchTime.IsZero() || p.lastFetchTime.Add(FETCH_INTERVAL).Before(time.Now()) {
		log.Println("Fetching new data for template")
		p.lastFetchTime = time.Now()
		p.cachedData = p.fetch(targetGoVersion)
		return p.cachedData
	}

	return p.cachedData
}

func (p *pluginsDataProvider) fetch(targetGoVersion string) PluginsData {
	data := PluginsData{}
	targetGolangV, err := semver.NewVersion(targetGoVersion)
	if err != nil {
		log.Printf("failed to parse target golang version: %s", targetGoVersion)
	}
	for _, plugin := range p.config.Plugins {
		devVersion := p.getGolangVersionOnRef(plugin, "develop")
		releasedVersion := p.getReleasedVersion(plugin)

		allBumped := false
		if targetGolangV != nil {
			pluginV, err := semver.NewVersion(releasedVersion)
			if err != nil {
				log.Printf("failed to parse image version for %s: %s", plugin.Name, err.Error())
			} else {
				if !targetGolangV.GreaterThan(pluginV) {
					allBumped = true
				}
			}
		}

		data.Plugins = append(data.Plugins, Plugin{
			Name:            plugin.Name,
			URL:             plugin.URL,
			DevVersion:      devVersion,
			ReleasedVersion: releasedVersion,
			AllBumped:       allBumped,
		})
	}
	return data
}

func (p *pluginsDataProvider) getReleasedVersion(plugin config.Plugin) string {
	publishedReleases, _, err := p.githubClient.Repositories.ListReleases(p.ctx, plugin.Owner, plugin.Repo, &github.ListOptions{PerPage: 1})
	if err != nil {
		return ""
	}
	if len(publishedReleases) < 1 {
		return ""

	}
	return p.getGolangVersionOnRef(plugin, publishedReleases[0].GetTagName())
}

func (p *pluginsDataProvider) getGolangVersionOnRef(plugin config.Plugin, ref string) string {
	docsContent, _, _, err := p.githubClient.Repositories.GetContents(p.ctx, plugin.Owner, plugin.Repo, "docs/go.version", &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return ""
	}

	docs, err := docsContent.GetContent()
	if err != nil {
		return ""
	}

	lines := strings.Split(docs, "\n")
	if len(lines) < 2 {
		return ""
	}

	return lines[1]
}
