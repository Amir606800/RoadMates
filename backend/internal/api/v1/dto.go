package v1

// DTOs belong to their API version, independently of domain and DB structs.
type IndexResponse struct {
	APIVersion string `json:"api_version"`
}
