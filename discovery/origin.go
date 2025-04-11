package discovery

import "github.com/lone-faerie/mqttop/internal/build"

// Origin implements the origin mapping for the discovery payload. This provides context to
// Home Assistant on the origin of the components.
type Origin struct {
	Name       string `json:"name"`
	SWVersion  string `json:"sw,omitempty"`
	SupportURL string `json:"url,omitempty"`
}

// NewOrigin returns the default Origin with the following values:
//   - Name: "mqttop"
//   - SWVersion: [build.Version]
//   - SupportURL: "https://github.com/lone-faerie/mqttop"
func NewOrigin() *Origin {
	return &Origin{
		Name:       "mqttop",
		SWVersion:  build.Version(),
		SupportURL: "https://" + build.Package(),
	}
}
