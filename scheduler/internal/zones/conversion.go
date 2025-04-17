package zones

// cloudRegionToPowerZoneMap defines the mapping from a specific cloud region
// to its corresponding power grid zone identifier.
var cloudRegionToPowerZoneMap = map[CloudRegion]PowerZone{
	{Provider: GCP, Name: "us-west2"}:             CAISO_NORTH,
	{Provider: GCP, Name: "us-east4"}:             PJM_DC,
	{Provider: GCP, Name: "europe-west3"}:         DE,
	{Provider: GCP, Name: "australia-southeast1"}: NEM_NSW,
}

// CloudRegionToPowerZone attempts to convert a CloudRegion struct to its corresponding PowerZone.
// It looks up the region in the predefined mapping.
// Returns the PowerZone and a boolean indicating if a mapping was found.
func CloudRegionToPowerZone(region CloudRegion) (PowerZone, bool) {
	powerZone, ok := cloudRegionToPowerZoneMap[region]
	return powerZone, ok
}

// CloudRegionStringToPowerZone is a convenience function that combines validation and conversion.
// It takes a user-provided string identifier (e.g., "aws:us-east-1"), validates it,
// and if valid, converts it to the corresponding PowerZone.
// Returns the PowerZone and a boolean indicating success.
func CloudRegionStringToPowerZone(identifier string) (PowerZone, bool) {
	cloudRegion, ok := GetCloudRegionFromString(identifier)
	if !ok {
		return PowerZone(""), false // Input string is not a valid/allowed cloud region identifier
	}
	return CloudRegionToPowerZone(cloudRegion)
}
