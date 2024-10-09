package host

import (
	"fmt"
)

func ServerPowerOff() {
	url := fmt.Sprintf("https://%s%s/Actions/ComputerSystem.Reset", CurrentHost.BMCAddress, RedfishPath)
	body := []byte(`{"ResetType": "ForceOff"}`)
	MakeRequest("POST", url, body)
}

func ServerPowerOn() {
	url := fmt.Sprintf("https://%s%s/Actions/ComputerSystem.Reset", CurrentHost.BMCAddress, RedfishPath)
	body := []byte(`{"ResetType": "On"}`)
	MakeRequest("POST", url, body)
}

func ServerRestart() {
	url := fmt.Sprintf("https://%s%s/Actions/ComputerSystem.Reset", CurrentHost.BMCAddress, RedfishPath)
	body := []byte(`{"ResetType": "ForceRestart"}`)
	MakeRequest("POST", url, body)
}

func ServerVirtualMediaInsert() {
	url := fmt.Sprintf("https://%s/redfish/v1/Managers/1/VirtualMedia/CD1/Actions/VirtualMedia.InsertMedia", CurrentHost.BMCAddress)
	body := []byte(fmt.Sprintf(`{"Image": "%s"}`, CurrentHost.ISOImage))
	MakeRequest("POST", url, body)
}

func ServerVirtualMediaEject() {
	url := fmt.Sprintf("https://%s/redfish/v1/Managers/1/VirtualMedia/CD1/Actions/VirtualMedia.EjectMedia", CurrentHost.BMCAddress)
	body := []byte(`{}`)
	MakeRequest("POST", url, body)
}

func ServerVirtualMediaStatus() {
	url := fmt.Sprintf("https://%s/redfish/v1/Managers/1/VirtualMedia/1", CurrentHost.BMCAddress)
	MakeRequest("GET", url, nil)
}

func ServerSetBootOnceCD() {
	url := fmt.Sprintf("https://%s%s", CurrentHost.BMCAddress, RedfishPath)
	body := []byte(`{"Boot":{ "BootSourceOverrideEnabled": "Once", "BootSourceOverrideTarget": "Cd", "BootSourceOverrideMode": "UEFI"}}`)
	MakeRequest("PATCH", url, body)
}

func ServerSetBootOncePXE() {
	url := fmt.Sprintf("https://%s%s", CurrentHost.BMCAddress, RedfishPath)
	body := []byte(`{"Boot":{ "BootSourceOverrideEnabled": "Once", "BootSourceOverrideTarget": "Pxe", "BootSourceOverrideMode": "UEFI"}}`)
	MakeRequest("PATCH", url, body)
}

func ServerCreateUser(username string, password string) {
	url := fmt.Sprintf("https://%s/redfish/v1/AccountService/Accounts", CurrentHost.BMCAddress)
	body := []byte(fmt.Sprintf(`{"UserName": "%s", "Password": "%s", "RoleId": "Administrator", "Enabled" : true}`, username, password))
	
	MakeRequest("POST", url, body)
}

func ServerEnableACL() {
	url := fmt.Sprintf("https://%s/redfish/v1/Managers/1/Oem/Supermicro/IPAccessControl", CurrentHost.BMCAddress)
	body := []byte(`{"ServiceEnabled": true}`)
	
	MakeRequest("PATCH", url, body)
}
func ServerCreateACL(address string, netmask string, policy string) {
	url := fmt.Sprintf("https://%s/redfish/v1/Managers/1/Oem/Supermicro/IPAccessControl/FilterRules", CurrentHost.BMCAddress)
	body := []byte(fmt.Sprintf(`{"Address": "%s", "PrefixLength": %s, "Policy": "%s"}`, address, netmask, policy))
	
	MakeRequest("POST", url, body)
}
