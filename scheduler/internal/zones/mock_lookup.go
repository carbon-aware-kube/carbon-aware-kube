package zones

// ZoneLookup defines the interface for looking up power zones.
// We define this here to allow mocking.
type ZoneLookup interface {
	GetPowerZone(zoneIdentifier string) (PowerZone, bool)
}

// MockStaticZoneLookup provides a simple map-based lookup for testing.
type MockStaticZoneLookup struct {
	zones map[string]PowerZone
}

// NewMockStaticZoneLookup creates a new mock lookup.
func NewMockStaticZoneLookup(zones map[string]PowerZone) *MockStaticZoneLookup {
	return &MockStaticZoneLookup{zones: zones}
}

// GetPowerZone implements the ZoneLookup interface for the mock.
func (m *MockStaticZoneLookup) GetPowerZone(zoneIdentifier string) (PowerZone, bool) {
	zone, ok := m.zones[zoneIdentifier]
	return zone, ok
}
