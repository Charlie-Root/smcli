package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Host struct {
	Name             string `yaml:"name"`
	BMCAddress       string `yaml:"bmc_address"`
	UsernamePassword string `yaml:"username_password"`
	ISOImage         string `yaml:"iso_image"`
}

type Config struct {
	Hosts []Host `yaml:"hosts"`
}

var ConfigData Config

func LoadConfig(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Error opening config file: %v", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&ConfigData)
	if err != nil {
		log.Fatalf("Error decoding config file: %v", err)
	}
}
