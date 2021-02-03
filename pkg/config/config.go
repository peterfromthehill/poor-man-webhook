package config

import (
	"encoding/json"
	"io/ioutil"

	corev1 "k8s.io/api/core/v1"
)

// Config options for the autocert admission controller.
type Config struct {
	Address                         string           `json:"address"`
	Service                         string           `json:"service"`
	Namespace                       string           `json:"namespace"`
	CAUrl                           string           `json:"caUrl"`
	EMail                           string           `json:"eMail"`
	LogFormat                       string           `json:"logFormat"`
	Iptables                        corev1.Container `json:"iptables"`
	Proxy                           corev1.Container `json:"proxy"`
	CertsVolume                     corev1.Volume    `json:"certsVolume"`
	RestrictCertificatesToNamespace bool             `json:"restrictCertificatesToNamespace"`
	ClusterDomain                   string           `json:"clusterDomain"`
	ServicePort                     ServicePort      `json:"servicePort"`
}

type ServicePort struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

func LoadConfig(file string) (*Config, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	cfg := Config{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetAddress returns the address set in the configuration, defaults to ":4443"
// if it's not specified.
func (c Config) GetAddress() string {
	if c.Address != "" {
		return c.Address
	}

	return ":4443"
}

// GetServiceName returns the service name set in the configuration, defaults to
// "autocert" if it's not specified.
func (c Config) GetServiceName() string {
	if c.Service != "" {
		return c.Service
	}

	return "autocert"
}

func (c Config) GetClusterDomain() string {
	if c.ClusterDomain != "" {
		return c.ClusterDomain
	}

	return "cluster.local"
}
