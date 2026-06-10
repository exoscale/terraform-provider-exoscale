package instance

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type NetworkInterface struct {
	NetworkID  string  `json:"network_id"`
	IPAddress  *string `json:"ip_address"`
	MacAddress string  `json:"mac_address"`
}

func NewNetworkInterface(raw any) (*NetworkInterface, error) {
	serializedRule, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	nif := NetworkInterface{}
	if err := json.Unmarshal(serializedRule, &nif); err != nil {
		tflog.Warn(context.Background(), err.Error())
		return nil, err
	}

	return &nif, nil
}

func (n NetworkInterface) ToInterface() (map[string]any, error) {
	serialized, err := json.Marshal(n)
	if err != nil {
		return nil, err
	}

	var nif map[string]any
	if err := json.Unmarshal(serialized, &nif); err != nil {
		return nil, err
	}

	return nif, nil
}
