package opensearch

// OpenSearch index mapping
const mapping = `{
	"mappings": {
		"properties": {
			"ptype": { "type": "keyword" },
			"v0": { "type": "keyword" },
			"v1": { "type": "keyword" },
			"v2": { "type": "keyword" },
			"v3": { "type": "keyword" },
			"v4": { "type": "keyword" },
			"v5": { "type": "keyword" }
		}
	}
}`

type Policy struct {
	PType string `json:"ptype"`
	V0    string `json:"v0"`
	V1    string `json:"v1"`
	V2    string `json:"v2,omitempty"`
	V3    string `json:"v3,omitempty"`
	V4    string `json:"v4,omitempty"`
	V5    string `json:"v5,omitempty"`
}
