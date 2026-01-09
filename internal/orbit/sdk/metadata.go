package sdk

// Metadata contains identity and version information for an orbit.
type Metadata struct {
	// ID is the unique identifier for this orbit.
	// Format: {vendor}.{name} (e.g., "orbita.wellness", "acme.pomodoro")
	ID string `json:"id"`

	// Name is the human-readable name of the orbit.
	Name string `json:"name"`

	// Version is the semantic version of the orbit.
	Version string `json:"version"`

	// Author is the creator or maintainer of the orbit.
	Author string `json:"author,omitempty"`

	// Description is a brief description of what the orbit does.
	Description string `json:"description,omitempty"`

	// License is the license under which the orbit is distributed.
	License string `json:"license,omitempty"`

	// Homepage is the URL to the orbit's documentation or homepage.
	Homepage string `json:"homepage,omitempty"`

	// Tags are searchable keywords for the orbit.
	Tags []string `json:"tags,omitempty"`

	// MinAPIVersion is the minimum Orbit SDK version required.
	MinAPIVersion string `json:"min_api_version,omitempty"`

	// RequiredOrbit is the parent orbit ID if this extends another orbit.
	// Used for marketplace add-ons that extend base orbits.
	RequiredOrbit string `json:"required_orbit,omitempty"`
}

// Validate checks that the metadata has all required fields.
func (m Metadata) Validate() error {
	if m.ID == "" {
		return ErrMissingID
	}
	if m.Name == "" {
		return ErrMissingName
	}
	if m.Version == "" {
		return ErrMissingVersion
	}
	return nil
}

// String returns a string representation of the metadata.
func (m Metadata) String() string {
	return m.ID + "@" + m.Version
}
