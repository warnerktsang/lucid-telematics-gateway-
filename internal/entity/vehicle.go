package entity

// Vehicle represents the vehicle identity carried by a per-vehicle JWT.
type Vehicle struct {
	VIN     string
	FleetID string
}

// GetVIN returns the vehicle identification number.
func (v Vehicle) GetVIN() string {
	return v.VIN
}

// GetFleetID returns the ID of the fleet the vehicle belongs to.
func (v Vehicle) GetFleetID() string {
	return v.FleetID
}
