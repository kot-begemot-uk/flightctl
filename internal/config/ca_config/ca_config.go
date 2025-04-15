package ca_config

type CAIdType int

const (
	InternalCA CAIdType = iota + 1
	AsyncInternalCA
)

type InternalCfg struct {
	CaCertFile         string `json:"caCertFile,omitempty"`
	CaKeyFile          string `json:"caKeyFile,omitempty"`
	SignerCertName     string `json:"signerCertName,omitempty"`
	CaSerialFile       string `json:"caSerialFile,omitempty"`
	CaCertValidityDays int    `json:"caCertValidityDays,omitempty"`
	CertStore          string `json:"certStore,omitempty"`
}

type Config struct {
	CAType                          CAIdType       `json:"type,omitempty"`
	AdminCommonName                 string         `json:"adminCommonName,omitempty"`
	ClientBootstrapCommonName       string         `json:"clientBootstrapCommonName,omitempty"`
	ClientBootstrapCertName         string         `json:"clientBootstrapCertName,omitempty"`
	ClientBootstrapCommonNamePrefix string         `json:"clientBootstrapCommonNamePrefix,omitempty"`
	ClientBootstrapValidityDays     int            `json:"clientBootStrapValidityDays,omitempty"`
	DeviceCommonNamePrefix          string         `json:"deviceCommonNamePrefix,omitempty"`
	InternalConfig                  *InternalCfg `json:"internalConfig,omitempty"`
	ServerCertValidityDays          int            `json:"serverCertValidityDays,omitempty"`
	ExtraAllowedPrefixes            []string       `json:"extraAllowedPrefixes,omitempty"`
}

func NewDefault(tempDir string) *Config {
	c := &Config{
		CAType:                          InternalCA,
		AdminCommonName:                 "flightctl-admin",
		ClientBootstrapCertName:         "client-enrollment",
		ClientBootstrapCommonName:       "client-enrollment",
		ClientBootstrapCommonNamePrefix: "client-enrollment-",
		ClientBootstrapValidityDays:     365,
		ServerCertValidityDays:          365,
		DeviceCommonNamePrefix:          "device:",
		InternalConfig: &InternalCfg{
			CaCertFile:         "ca.crt",
			CaKeyFile:          "ca.key",
			CaCertValidityDays: 3650,
			SignerCertName:     "ca",
			CertStore:        tempDir,
		},
	}
	return c
}
