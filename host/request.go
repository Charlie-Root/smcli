package host

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func MakeRequest(method, url string, body []byte) []byte {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		log.Fatal(err)
	}
	authParts := strings.SplitN(CurrentHost.UsernamePassword, ":", 2)
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
		var success struct {
			Message string `json:"message"`
		}
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
		var errorResp struct {
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
