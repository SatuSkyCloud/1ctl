package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TalosConfig represents a Talos configuration
type TalosConfig struct {
	ID          uuid.UUID `json:"id"`
	MachineID   uuid.UUID `json:"machine_id"`
	ClusterName string    `json:"cluster_name"`
	Role        string    `json:"role"`
	Version     string    `json:"version"`
	ConfigData  string    `json:"config_data,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	AppliedAt   time.Time `json:"applied_at,omitempty"`
}

// TalosConfigHistory represents config history entry
type TalosConfigHistory struct {
	ID        uuid.UUID `json:"id"`
	MachineID uuid.UUID `json:"machine_id"`
	Version   string    `json:"version"`
	AppliedAt time.Time `json:"applied_at"`
	AppliedBy string    `json:"applied_by"`
	Status    string    `json:"status"`
}

// TalosNetworkInfo represents network info for a machine
type TalosNetworkInfo struct {
	MachineID  uuid.UUID          `json:"machine_id"`
	Hostname   string             `json:"hostname"`
	Addresses  []string           `json:"addresses"`
	Interfaces []NetworkInterface `json:"interfaces"`
	DefaultGW  string             `json:"default_gateway"`
	DNS        []string           `json:"dns_servers"`
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name      string   `json:"name"`
	MAC       string   `json:"mac"`
	Addresses []string `json:"addresses"`
	State     string   `json:"state"`
}

// GenerateTalosConfigRequest represents config generation request
type GenerateTalosConfigRequest struct {
	MachineID   string `json:"machine_id"`
	ClusterName string `json:"cluster_name"`
	Role        string `json:"role"`
}

// ApplyTalosConfigRequest represents config apply request
type ApplyTalosConfigRequest struct {
	MachineID  string `json:"machine_id"`
	ConfigData string `json:"config_data"`
}

// GenerateTalosConfig generates a Talos configuration
func GenerateTalosConfig(req GenerateTalosConfigRequest) (*TalosConfig, error) {
	var resp apiResponse
	err := makeRequest("POST", "/talos/generate-config", req, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var config TalosConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal talos config: %s", err.Error()), nil)
	}
	return &config, nil
}

// ApplyTalosConfig applies a Talos configuration to a machine
func ApplyTalosConfig(req ApplyTalosConfigRequest) error {
	return makeRequest("POST", "/talos/apply-config", req, nil)
}

// GetTalosConfigHistory gets config history for a machine
func GetTalosConfigHistory(machineID string) ([]TalosConfigHistory, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/talos/config-history/%s", machineID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var history []TalosConfigHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal config history: %s", err.Error()), nil)
	}
	return history, nil
}

// GetTalosNetworkInfo gets network info for a machine
func GetTalosNetworkInfo(machineID string) (*TalosNetworkInfo, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/talos/network-info/%s", machineID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var info TalosNetworkInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal network info: %s", err.Error()), nil)
	}
	return &info, nil
}
