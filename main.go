package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/cloudfoundry-incubator/golang-bump-progress/config"
	"github.com/cloudfoundry-incubator/golang-bump-progress/dataprovider"
	"github.com/cloudfoundry-incubator/golang-bump-progress/version"
	"github.com/google/go-github/v54/github"
	"golang.org/x/oauth2"
)

func main() {
	tmpl := template.Must(template.ParseFiles("templates/table.html"))
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("failed to load config: %s", err.Error())
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	ctx := context.Background()
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)
	boshPackageVersion := version.NewBoshPackageVersion(ctx, githubClient)
	err = boshPackageVersion.PopulateCache()
	if err != nil {
		log.Fatalf("failed to warm up cache: %s", err.Error())
	}

	githubVersion := version.NewGithubVersion(ctx, githubClient, boshPackageVersion)
	tasVersion := version.NewTasVersion(ctx, githubClient)
	err = tasVersion.Fetch("main")
	if err != nil {
		log.Printf("failed to get TAS versions: %s", err.Error())
	}
	templateDataProvider := dataprovider.NewTemplateDataProvider(githubVersion, tasVersion, cfg)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := templateDataProvider.Get()
		tmpl.Execute(w, data)
	})

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
