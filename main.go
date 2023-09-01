package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/cloudfoundry-incubator/golang-bump-progress/config"
	"github.com/google/go-github/v54/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

type Release struct {
	Name         string
	URL          string
	VersionOnDev string
}

type TemplateData struct {
	Releases []Release
}

type PackageSpec struct {
	Name        string `yaml:"name"`
	Fingerprint string `yaml:"fingerprint"`
}

const GOLANG_BOSH_RELEASE_OWNER = "cloudfoundry"
const GOLANG_BOSH_RELEASE_REPO = "bosh-package-golang-release"

func main() {
	tmpl := template.Must(template.ParseFiles("templates/table.html"))
	var cfg config.Config
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal("failed to read config file")
	}
	err = json.Unmarshal([]byte(configFile), &cfg)
	if err != nil {
		log.Fatal("failed to parse config")
	}

	githubToken := os.Getenv("GITHUB_TOKEN")

	githubVersionFetcher := version.NewGithubVersionFetcher(

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)

	data := TemplateData{
		Releases: []Release{},
	}
	for _, rel := range cfg.Releases {

		versionFetcher.GetDevelopVersion(rel)
		url, err := url.Parse(rel.URL)
		if err != nil {
			log.Fatalf("failed to parse URL: %s", err.Error())
		}
		parts := strings.Split(strings.TrimLeft(url.Path, "/"), "/")
		developSpecContent, _, _, err := githubClient.Repositories.GetContents(ctx, parts[0], parts[1], fmt.Sprintf("packages/%s/spec.lock", rel.GolangPackage), &github.RepositoryContentGetOptions{Ref: "develop"})
		if err != nil {
			log.Fatalf("failed to get github info: %s", err.Error())
		}
		developSpec, err := developSpecContent.GetContent()
		if err != nil {
			log.Fatalf("failed to get content: %s", err.Error())
		}

		var packageSpec PackageSpec
		err = yaml.Unmarshal([]byte(developSpec), &packageSpec)
		if err != nil {
			log.Fatalf("failed to parse package spec: %s", err.Error())
		}

		query := fmt.Sprintf("%s repo:%s/%s", packageSpec.Fingerprint, GOLANG_BOSH_RELEASE_OWNER, GOLANG_BOSH_RELEASE_REPO)

		searchResult, _, err := githubClient.Search.Code(ctx, query, nil)
		if err != nil {
			log.Fatalf("failed to search: %s", err.Error())
		}
		if searchResult.GetTotal() < 1 {
			log.Fatal("did not find the fingerprint code")
		}

		codeResult := searchResult.CodeResults[0]
		refURL := codeResult.GetHTMLURL()
		re := regexp.MustCompile(`.+/blob/(\w+)/.+`)
		matches := re.FindSubmatch([]byte(refURL))
		if len(matches) < 2 {
			log.Fatal("failed to parse ref")
		}

		ref := string(matches[1])

		versionContent, _, _, err := githubClient.Repositories.GetContents(ctx, GOLANG_BOSH_RELEASE_OWNER, GOLANG_BOSH_RELEASE_REPO, fmt.Sprintf("packages/%s/version", rel.GolangPackage), &github.RepositoryContentGetOptions{Ref: ref})
		if err != nil {
			log.Fatalf("failed to get github info: %s", err.Error())
		}
		version, err := versionContent.GetContent()
		if err != nil {
			log.Fatalf("failed to get content: %s", err.Error())
		}

		data.Releases = append(data.Releases, Release{
			Name:         rel.Name,
			URL:          rel.URL,
			VersionOnDev: version,
		})
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%#v\n", data)
		tmpl.Execute(w, data)
	})

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
