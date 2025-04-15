package zones

// PowerZone represents a power grid zone identifier (e.g., "US-CAL-CISO").
// Renamed from Zone for clarity.
type PowerZone string

const (
	CAISO_NORTH PowerZone = "CAISO_NORTH"
)

// CloudProvider represents a cloud service provider identifier.
type CloudProvider string

const (
	AWS   CloudProvider = "aws"
	GCP   CloudProvider = "gcp"
	Azure CloudProvider = "azure"
)

// CloudRegion represents a specific region within a cloud provider.
// This struct holds structured information about a cloud region.
type CloudRegion struct {
	Provider CloudProvider
	Name     string // e.g., "us-east-1", "us-central1"
}

// AllowedCloudRegions defines the canonical set of cloud regions supported by the API.
// The key is the string identifier users will provide (e.g., "gcp:us-west2").
// Using a map allows quick validation of user input.
var AllowedCloudRegions = map[string]CloudRegion{
	"gcp:us-west2": {Provider: GCP, Name: "us-west2"},
	// Add more supported regions here as needed
}

// IsValidCloudRegion checks if a given string identifier corresponds to a supported cloud region.
func IsValidCloudRegion(identifier string) bool {
	_, ok := AllowedCloudRegions[identifier]
	return ok
}

// GetCloudRegionFromString attempts to retrieve the CloudRegion struct for a given identifier.
// Returns the struct and a boolean indicating if it was found.
func GetCloudRegionFromString(identifier string) (CloudRegion, bool) {
	region, ok := AllowedCloudRegions[identifier]
	return region, ok
}
