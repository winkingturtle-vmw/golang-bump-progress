package config

type Release struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	GolangPackage string `json:"golang_package"`
}

type Config struct {
	Releases []Release `json:"releases"`
}
