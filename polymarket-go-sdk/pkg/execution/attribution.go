package execution

import "strings"

const (
	HeaderAttributionBuilder = "X-Attribution-Builder"
	HeaderAttributionFunder  = "X-Attribution-Funder"
	HeaderAttributionSource  = "X-Attribution-Source"
)

// Attribution carries builder/funder/source metadata that upstream services can pass through.
type Attribution struct {
	Builder string `json:"builder,omitempty"`
	Funder  string `json:"funder,omitempty"`
	Source  string `json:"source,omitempty"`
}

// NormalizeAttribution returns a canonical form suitable for cross-service propagation.
func NormalizeAttribution(in Attribution) Attribution {
	return Attribution{
		Builder: strings.ToLower(strings.TrimSpace(in.Builder)),
		Funder:  strings.ToLower(strings.TrimSpace(in.Funder)),
		Source:  strings.ToLower(strings.TrimSpace(in.Source)),
	}
}

// HeaderMap converts attribution fields to HTTP header map.
func (a Attribution) HeaderMap() map[string]string {
	n := NormalizeAttribution(a)
	out := map[string]string{}
	if n.Builder != "" {
		out[HeaderAttributionBuilder] = n.Builder
	}
	if n.Funder != "" {
		out[HeaderAttributionFunder] = n.Funder
	}
	if n.Source != "" {
		out[HeaderAttributionSource] = n.Source
	}
	return out
}
