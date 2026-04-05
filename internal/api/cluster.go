package api

// ZoneOption represents an available deployment zone.
type ZoneOption struct {
	Value     string `json:"value"`
	Label     string `json:"label"`
	ClusterID string `json:"cluster_id"`
}

// ClusterInfo represents a cluster in the registry.
type ClusterInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Region      string `json:"region"`
	Zone        string `json:"zone"`
	Endpoint    string `json:"endpoint"`
	Priority    int    `json:"priority"`
	IsDefault   bool   `json:"is_default"`
	Enabled     bool   `json:"enabled"`
	Healthy     bool   `json:"healthy"`
}

// GetAvailableZones fetches available deployment zones from the backend.
func GetAvailableZones() ([]ZoneOption, error) {
	var zones []ZoneOption
	err := makeRequest("GET", "/clusters/zones", nil, &zones)
	if err != nil {
		return nil, err
	}
	return zones, nil
}

// GetClusters fetches all enabled clusters from the backend.
func GetClusters() ([]ClusterInfo, error) {
	var clusters []ClusterInfo
	err := makeRequest("GET", "/clusters", nil, &clusters)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}
