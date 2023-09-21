package dataprovider

import (
	"log"
	"time"

	"github.com/cloudfoundry-incubator/golang-bump-progress/config"
)

type Image struct {
	Name      string
	URL       string
	Version   string
	AllBumped bool
}

type ImagesData struct {
	Images []Image
}

type imagesDataProvider struct {
	config        config.Config
	lastFetchTime time.Time
	cachedData    ImagesData
}

func NewImagesDataProvider(cfg config.Config) *imagesDataProvider {
	return &imagesDataProvider{
		config: cfg,
	}
}

func (p *imagesDataProvider) Get() ImagesData {
	if p.lastFetchTime.IsZero() || p.lastFetchTime.Add(FETCH_INTERVAL).Before(time.Now()) {
		log.Println("Fetching new data for template")
		p.lastFetchTime = time.Now()
		p.cachedData = p.fetch()
		return p.cachedData
	}

	return p.cachedData
}

func (p *imagesDataProvider) fetch() ImagesData {
	data := ImagesData{}
	for _, image := range p.config.Images {
		var allBumped bool
		var version string

		data.Images = append(data.Images, Image{
			Name:      image.Name,
			URL:       image.URL,
			Version:   version,
			AllBumped: allBumped,
		})
	}
	return data
}
