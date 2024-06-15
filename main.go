package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
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

type ErrorResponse struct {
	Error struct {
		Code            string `json:"code"`
		Message         string `json:"message"`
		MessageExtended []struct {
			MessageId         string   `json:"MessageId"`
			Severity          string   `json:"Severity"`
			Resolution        string   `json:"Resolution"`
			Message           string   `json:"Message"`
			MessageArgs       []string `json:"MessageArgs"`
			RelatedProperties []string `json:"RelatedProperties"`
		} `json:"@Message.ExtendedInfo"`
	} `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

var (
	config      Config
	currentHost Host
	redfishPath string
	verbose     bool
)

func main() {
	loadConfig("inventory.yaml")

	var rootCmd = &cobra.Command{Use: "sm-cli"}
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	var powerCmd = &cobra.Command{
		Use:   "power [on|off|restart]",
		Short: "Manage power",
		Args:  cobra.ExactArgs(2),
		Run:   powerCommand,
	}
	var mediaCmd = &cobra.Command{
		Use:   "media [insert|eject|status]",
		Short: "Manage virtual media",
		Args:  cobra.ExactArgs(2),
		Run:   mediaCommand,
	}
	var bootCmd = &cobra.Command{
		Use:   "boot [cd|pxe]",
		Short: "Set boot order",
		Args:  cobra.ExactArgs(2),
		Run:   bootCommand,
	}

	rootCmd.AddCommand(powerCmd)
	rootCmd.AddCommand(mediaCmd)
	rootCmd.AddCommand(bootCmd)
	rootCmd.Execute()
}

func loadConfig(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Error opening config file: %v", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalf("Error decoding config file: %v", err)
	}
}

func selectHost(name string) {
	for _, host := range config.Hosts {
		if host.Name == name {
			currentHost = host
			return
		}
	}
	log.Fatalf("Host with name %s not found", name)
}

func getRedfishPath() {
	url := fmt.Sprintf("https://%s/redfish/v1/Systems", currentHost.BMCAddress)
	respBody := makeRequest("GET", url, nil)

	var data map[string]interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		log.Fatal(err)
	}

	if members, ok := data["Members"].([]interface{}); ok && len(members) > 0 {
		if member, ok := members[0].(map[string]interface{}); ok {
			redfishPath = member["@odata.id"].(string)
		}
	}
}

func powerCommand(cmd *cobra.Command, args []string) {
	action := args[0]
	host := args[1]

	selectHost(host)
	getRedfishPath()

	switch action {
	case "on":
		serverPowerOn()
	case "off":
		serverPowerOff()
	case "restart":
		serverRestart()
	default:
		fmt.Println("Unknown power command:", action)
	}
}

func mediaCommand(cmd *cobra.Command, args []string) {
	action := args[0]
	host := args[1]

	selectHost(host)
	getRedfishPath()

	switch action {
	case "insert":
		serverVirtualMediaInsert()
	case "eject":
		serverVirtualMediaEject()
	case "status":
		serverVirtualMediaStatus()
	default:
		fmt.Println("Unknown media command:", action)
	}
}

func bootCommand(cmd *cobra.Command, args []string) {
	action := args[0]
	host := args[1]

	selectHost(host)
	getRedfishPath()

	switch action {
	case "cd":
		serverSetBootOnceCD()
	case "pxe":
		serverSetBootOncePXE()
	default:
		fmt.Println("Unknown boot command:", action)
	}
}

func serverPowerOff() {
	url := fmt.Sprintf("https://%s%s/Actions/ComputerSystem.Reset", currentHost.BMCAddress, redfishPath)
	body := []byte(`{"ResetType": "ForceOff"}`)
	makeRequest("POST", url, body)
}

func serverPowerOn() {
	url := fmt.Sprintf("https://%s%s/Actions/ComputerSystem.Reset", currentHost.BMCAddress, redfishPath)
	body := []byte(`{"ResetType": "On"}`)
	makeRequest("POST", url, body)
}

func serverRestart() {
	url := fmt.Sprintf("https://%s%s/Actions/ComputerSystem.Reset", currentHost.BMCAddress, redfishPath)
	body := []byte(`{"ResetType": "ForceRestart"}`)
	makeRequest("POST", url, body)
}

func serverVirtualMediaInsert() {
	url := fmt.Sprintf("https://%s/redfish/v1/Managers/1/VirtualMedia/CD1/Actions/VirtualMedia.InsertMedia", currentHost.BMCAddress)
	body := []byte(fmt.Sprintf(`{"Image": "%s"}`, currentHost.ISOImage))
	makeRequest("POST", url, body)
}

func serverVirtualMediaEject() {
	url := fmt.Sprintf("https://%s/redfish/v1/Managers/1/VirtualMedia/CD1/Actions/VirtualMedia.EjectMedia", currentHost.BMCAddress)
	body := []byte(`{}`)
	makeRequest("POST", url, body)
}

func serverVirtualMediaStatus() {
	url := fmt.Sprintf("https://%s/redfish/v1/Managers/1/VirtualMedia/1", currentHost.BMCAddress)
	makeRequest("GET", url, nil)
}

func serverSetBootOnceCD() {
	url := fmt.Sprintf("https://%s%s", currentHost.BMCAddress, redfishPath)
	body := []byte(`{"Boot":{ "BootSourceOverrideEnabled": "Once", "BootSourceOverrideTarget": "Cd", "BootSourceOverrideMode": "UEFI"}}`)
	makeRequest("PATCH", url, body)
}

func serverSetBootOncePXE() {
	url := fmt.Sprintf("https://%s%s", currentHost.BMCAddress, redfishPath)
	body := []byte(`{"Boot":{ "BootSourceOverrideEnabled": "Once", "BootSourceOverrideTarget": "Pxe", "BootSourceOverrideMode": "UEFI"}}`)
	makeRequest("PATCH", url, body)
}

func makeRequest(method, url string, body []byte) []byte {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		log.Fatal(err)
	}
	authParts := strings.SplitN(currentHost.UsernamePassword, ":", 2)
	req.SetBasicAuth(authParts[0], authParts[1])
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var success SuccessResponse
		if err := json.Unmarshal(respBody, &success); err == nil {
			if verbose {
				fmt.Println("Success:", success.Message)
			} 
		} else {
			if verbose {
				fmt.Println("Success:", string(respBody))
			} 
		}
	} else {
		var errorResp ErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			fmt.Printf("Error: %s - %s\n", errorResp.Error.Code, errorResp.Error.Message)
			for _, info := range errorResp.Error.MessageExtended {
				fmt.Printf("  %s: %s\n", info.Severity, info.Message)
				if info.Resolution != "" {
					fmt.Printf("    Resolution: %s\n", info.Resolution)
				}
			}
		} else {
			fmt.Printf("Error: %s\n", resp.Status)
			fmt.Println(string(respBody))
		}
	}
	
	return respBody
}
