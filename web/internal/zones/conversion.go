package zones

// cloudRegionToPowerZoneMap defines the mapping from a specific cloud region
// to its corresponding power grid zone identifier.
// --- IMPORTANT: Populate this map with the correct PowerZone for each CloudRegion ---
// You'll need to determine the correct power grid zone for each cloud region you support.
// The PowerZone values are examples and likely need correction based on actual grid data.
var cloudRegionToPowerZoneMap = map[CloudRegion]PowerZone{
	{Provider: GCP, Name: "us-west2"}: CAISO_NORTH, // Example mapping
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
		return "", false // Input string is not a valid/allowed cloud region identifier
	}
	return CloudRegionToPowerZone(cloudRegion)
}
