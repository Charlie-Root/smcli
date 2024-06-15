package host

import (
	"encoding/json"
	"fmt"
	"log"
)

func GetRedfishPath() {
	url := fmt.Sprintf("https://%s/redfish/v1/Systems", CurrentHost.BMCAddress)
	respBody := MakeRequest("GET", url, nil)

	var data map[string]interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		log.Fatal(err)
	}

	if members, ok := data["Members"].([]interface{}); ok && len(members) > 0 {
		if member, ok := members[0].(map[string]interface{}); ok {
			RedfishPath = member["@odata.id"].(string)
		}
	}
}
