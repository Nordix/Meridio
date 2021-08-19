package config

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

const (
	GatewayConfigKey = "gateways"
	VipsConfigKey    = "vips"
)

type GatewayConfig struct {
	Gateways []Gateway `yaml:"items"`
}

type VipConfig struct {
	Vips []Vip `yaml:"items"`
}

type Gateway struct {
	Name     string `yaml:"name"`
	Address  string `yaml:"address"`
	ASN      uint16 `yaml:"asn"`
	IPFamily string `yaml:"ip-family"`
	BFD      bool   `yaml:"bfd"`
	Protocol string `yaml:"protocol"`
	HoldTime uint   `yaml:"hold-time"`
}

type Vip struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
}

// Input configmap.Data
func UnmarshalConfig(data map[string]string) (*GatewayConfig, *VipConfig, error) {
	gw, errg := UnmarshalGatewayConfig(data[GatewayConfigKey])
	vip, errv := UnmarshalVipConfig(data[VipsConfigKey])
	if errg != nil || errv != nil {
		return nil, nil, fmt.Errorf("%s %s", errg, errv)
	}
	return gw, vip, nil
}

// Input is the value of GatewayConfigKey,
// returned as GatewayConfig struct
func UnmarshalGatewayConfig(c string) (*GatewayConfig, error) {
	config := &GatewayConfig{}
	err := yaml.Unmarshal([]byte(c), &config)
	return config, err
}

// Input is the value of VipsConfigKey,
// returned as VipConfig struct
func UnmarshalVipConfig(c string) (*VipConfig, error) {
	config := &VipConfig{}
	err := yaml.Unmarshal([]byte(c), &config)
	return config, err
}

// Input is VipConfig struct.
// Return a map with key as vip names, values of the key are
func MakeMapFromVipList(config *VipConfig) map[string]Vip {
	list := config.Vips
	ret := make(map[string]Vip)
	for _, item := range list {
		ret[item.Name] = item
	}
	return ret
}

// Input is VipConfig struct.
// Return a map with key as vip names.
func MakeMapFromGWList(config *GatewayConfig) map[string]Gateway {
	list := config.Gateways
	ret := make(map[string]Gateway)
	for _, item := range list {
		ret[item.Name] = item
	}
	return ret
}
