package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/viper"
)

type Config struct {
	// TODO: Separate config for signing node and for full node
	BtcNodeConfig   BtcConfig    `mapstructure:"btc-config"`
	BtcSignerConfig BtcConfig    `mapstructure:"btc-signer-config"`
	Server          ServerConfig `mapstructure:"server-config"`
}

func DefaultConfig() *Config {
	return &Config{
		BtcNodeConfig:   *DefaultBtcConfig(),
		BtcSignerConfig: *DefaultBtcConfig(),
		Server:          *DefaultServerConfig(),
	}
}

type ParsedConfig struct {
	BtcNodeConfig   *ParsedBtcConfig
	BtcSignerConfig *ParsedBtcConfig
	ServerConfig    *ParsedServerConfig
}

func (cfg *Config) Parse() (*ParsedConfig, error) {
	btcConfig, err := cfg.BtcNodeConfig.Parse()
	if err != nil {
		return nil, err
	}

	btcSignerConfig, err := cfg.BtcSignerConfig.Parse()

	if err != nil {
		return nil, err
	}

	serverConfig, err := cfg.Server.Parse()

	if err != nil {
		return nil, err
	}

	return &ParsedConfig{
		BtcNodeConfig:   btcConfig,
		BtcSignerConfig: btcSignerConfig,
		ServerConfig:    serverConfig,
	}, nil
}

const defaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

# There are two btc related configs
# 1. [btc-config] is config for btc full node which should have transaction indexing
# enabled. This node should be synced and can be open to the public.
# 2. [btc-signer-config] is config for bitcoind daemon which should have only
# wallet functionality, it should run in separate network. This bitcoind instance
# will be used to sign psbt's
[btc-config]
# Btc node host
host = "{{ .BtcNodeConfig.Host }}"
# Btc node user
user = "{{ .BtcNodeConfig.User }}"
# Btc node password
pass = "{{ .BtcNodeConfig.Pass }}"
# Btc network (testnet3|mainnet|regtest|simnet|signet)
network = "{{ .BtcNodeConfig.Network }}"

[btc-signer-config]
# Btc node host
host = "{{ .BtcSignerConfig.Host }}"
# TODO: consider reading user/pass from command line
# Btc node user
user = "{{ .BtcSignerConfig.User }}"
# Btc node password
pass = "{{ .BtcSignerConfig.Pass }}"
# Btc network (testnet3|mainnet|regtest|simnet|signet)
network = "{{ .BtcSignerConfig.Network }}"

[server-config]
# The address to listen on
host = "{{ .Server.Host }}"

# The port to listen on
port = {{ .Server.Port }}

# Read timeout in seconds
read-timeout = {{ .Server.ReadTimeout }}

# Write timeout in seconds
write-timeout = {{ .Server.WriteTimeout }}

# Idle timeout in seconds
idle-timeout = {{ .Server.IdleTimeout }}

`

var configTemplate *template.Template

func init() {
	var err error
	tmpl := template.New("configFileTemplate").Funcs(template.FuncMap{
		"StringsJoin": strings.Join,
	})
	if configTemplate, err = tmpl.Parse(defaultConfigTemplate); err != nil {
		panic(err)
	}
}

func writeConfigToFile(configFilePath string, config *Config) error {
	var buffer bytes.Buffer

	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	return os.WriteFile(configFilePath, buffer.Bytes(), 0o600)
}

func WriteConfigToFile(pathToConfFile string, conf *Config) error {
	dirPath, _ := filepath.Split(pathToConfFile)

	if _, err := os.Stat(pathToConfFile); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return fmt.Errorf("couldn't make config: %v", err)
		}

		if err := writeConfigToFile(pathToConfFile, conf); err != nil {
			return fmt.Errorf("could config to the file: %v", err)
		}
	}
	return nil
}

func fileNameWithoutExtension(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

func GetConfig(pathToConfFile string) (*Config, error) {
	dir, file := filepath.Split(pathToConfFile)
	configName := fileNameWithoutExtension(file)
	viper.SetConfigName(configName)
	viper.AddConfigPath(dir)
	viper.SetConfigType("toml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	conf := DefaultConfig()
	if err := viper.Unmarshal(conf); err != nil {
		return nil, err
	}

	return conf, nil
}
