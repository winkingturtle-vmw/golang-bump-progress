package version

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudfoundry-incubator/golang-bump-progress/config"
	"github.com/google/go-github/v54/github"
	"gopkg.in/yaml.v2"
)

type githubVersion struct {
	githubClient       *github.Client
	boshPackageVersion *boshPackageVersion

	ctx context.Context
}

func NewGithubVersion(ctx context.Context, githubClient *github.Client, boshPackageVersion *boshPackageVersion) *githubVersion {
	return &githubVersion{
		githubClient:       githubClient,
		boshPackageVersion: boshPackageVersion,
		ctx:                ctx,
	}
}

func (f *githubVersion) GetDevelopVersion(release config.Release) (string, error) {
	return f.getGolangVersionOnRef(release, "develop")
}

func (f *githubVersion) GetReleasedVersion(release config.Release) (string, error) {
	publishedReleases, _, err := f.githubClient.Repositories.ListReleases(f.ctx, release.Owner, release.Repo, &github.ListOptions{PerPage: 1})
	if err != nil {
		return "", err
	}
	if len(publishedReleases) < 1 {
		return "", errors.New("no results for published releases")

	}
	return f.getGolangVersionOnRef(release, publishedReleases[0].GetTagName())
}

func (f *githubVersion) GetFirstReleasedMinorVersion(release config.Release, releasedVersion string) (string, error) {
	println("releasedVersion", releasedVersion)
	releasedVersionMajorMinor := majorMinor(releasedVersion)
	println("minor, major", releasedVersionMajorMinor)
	publishedReleases, _, err := f.githubClient.Repositories.ListReleases(f.ctx, release.Owner, release.Repo, &github.ListOptions{PerPage: 5})
	if err != nil {
		return "", err
	}
	if len(publishedReleases) < 1 {
		return "", errors.New("no results for published releases")
	}

	var firstReleasedMinorVersion string
	for _, publishedRelease := range publishedReleases {
		golangVersion, err := f.getGolangVersionOnRef(release, publishedRelease.GetTagName())
		if err != nil {
			return "", err
		}
		fmt.Printf("release: %s, golang version: %s, commit: %s\n", publishedRelease.GetName(), golangVersion, publishedRelease.GetTagName())
		if majorMinor(golangVersion) == releasedVersionMajorMinor {
			firstReleasedMinorVersion = publishedRelease.GetName()
		} else {
			return firstReleasedMinorVersion, nil
		}
	}
	return "", errors.New("failed to find first min version")
}

func (f *githubVersion) getGolangVersionOnRef(release config.Release, ref string) (string, error) {
	developSpecContent, _, _, err := f.githubClient.Repositories.GetContents(f.ctx, release.Owner, release.Repo, fmt.Sprintf("packages/%s/spec.lock", release.GolangPackage), &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return "", err
	}
	developSpec, err := developSpecContent.GetContent()
	if err != nil {
		return "", err
	}

	var packageSpec PackageSpec
	err = yaml.Unmarshal([]byte(developSpec), &packageSpec)
	if err != nil {
		return "", err
	}

	fmt.Printf("getting version for fingerprint: %s\n", packageSpec.Fingerprint)

	return f.boshPackageVersion.GetFingerprintVersion(packageSpec.Fingerprint, release.GolangPackage)
}

func majorMinor(version string) string {
	parts := strings.Split(version, ".")
	return fmt.Sprintf("%s.%s", parts[0], parts[1])
}
