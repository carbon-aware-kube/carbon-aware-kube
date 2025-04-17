package zones

// SimpleZoneLookup provides a basic implementation of the ZoneLookup interface.
// It uses the existing CloudRegionStringToPowerZone logic.
type SimpleZoneLookup struct{}

// NewSimpleZoneLookup creates a new SimpleZoneLookup.
func NewSimpleZoneLookup() *SimpleZoneLookup {
	return &SimpleZoneLookup{}
}

// GetPowerZone implements the ZoneLookup interface.
func (l *SimpleZoneLookup) GetPowerZone(zoneIdentifier string) (PowerZone, bool) {
	// Use the conversion function which handles validation and mapping
	return CloudRegionStringToPowerZone(zoneIdentifier)
}

// Compile-time check to ensure SimpleZoneLookup implements ZoneLookup
var _ ZoneLookup = (*SimpleZoneLookup)(nil)
