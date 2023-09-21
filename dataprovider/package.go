package dataprovider // import "github.com/cloudfoundry-incubator/golang-bump-progress/dataprovider"

import (
	"time"

	"github.com/cloudfoundry-incubator/golang-bump-progress/version"
)

const (
	FETCH_INTERVAL        = time.Minute
	TARGET_GOLANG_VERSION = "1.21.0" // TODO: pull this from tas-runtime/go.version once implemented
)

type TemplateData struct {
	GolangVersion string
}

var BaseTemplateData = TemplateData{
	GolangVersion: version.MajorMinor(TARGET_GOLANG_VERSION),
}
