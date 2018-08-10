package aurora

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// ClientInterface is an interface for aurora api clients
type ClientInterface interface {
	// Aurora maintenance things needed:
	//GetMaintenanceStatus(map[string]string) (*MaintenanceStatusResponse, error)
	StartMaintenance(map[string]string) error
	EndMaintenance(map[string]string) error
	DrainHosts(map[string]string) error
}

// Client implements a client for aurora api
type Client struct {
	APIUrl string // url for the /apibeta path
}

// MaintenanceStatusResponse is returned from GetMaintenanceStatus()
/* {
    "responseCode": "OK",
    "serverInfo": {
        "clusterName": "dub-test",
        "thriftAPIVersion": 3,
        "statsUrlPrefix": ""
    },
    "result": {
        "maintenanceStatusResult": {
            "statuses": [
                {
                    "host": "10.19.65.11",
                    "mode": "SCHEDULED"
                }
            ]
        }
    },
    "details": []
} */
type MaintenanceStatusResponse struct {
	Result MaintenanceStatusResult `json:"result"`
}

// MaintenanceStatusResult is the result contained in a MaintenanceStatusResponse
type MaintenanceStatusResult struct {
	ResultStatuses MaintenanceStatuses `json:"maintenanceStatusResult"`
}

// MaintenanceStatuses contains list of individual []MaintenanceHostStatus
type MaintenanceStatuses struct {
	Status []MaintenanceHostStatus `json:"statuses"`
}

// MaintenanceHostStatus contains a Host and its Mode (Maintenance Mode) - one of NONE, SCHEDULED, DRAINING, DRAINED
type MaintenanceHostStatus struct {
	Host string `json:"host"`
	Mode string `json:"mode"`
}

// StartMaintenanceResponse is returned from StartMaintenance()
/* {
    "responseCode": "OK",
    "serverInfo": {
        "clusterName": "dub-test",
        "thriftAPIVersion": 3,
        "statsUrlPrefix": ""
    },
    "result": {
        "startMaintenanceResult": {
            "statuses": [
                {
                    "host": "10.19.65.25",
                    "mode": "SCHEDULED"
                }
            ]
        }
    },
    "details": []
} */
type StartMaintenanceResponse struct {
	Result StartMaintenanceResult `json:"results"`
}

// StartMaintenanceResult is the result contained in a StartMaintenanceResponse
type StartMaintenanceResult struct {
	ResultStatuses StartMaintenanceStatuses `json:"startMaintenanceResult"`
}

// StartMaintenanceStatuses contains list of individual []MaintenanceHostStatus
type StartMaintenanceStatuses struct {
	Status []MaintenanceHostStatus `json:"statuses"`
}

// DrainHostsResponse is returned from DrainHosts()
/*
{
    "responseCode": "OK",
    "serverInfo": {
        "clusterName": "dub-test",
        "thriftAPIVersion": 3,
        "statsUrlPrefix": ""
    },
    "result": {
        "drainHostsResult": {
            "statuses": [
                {
                    "host": "10.19.65.11",
                    "mode": "DRAINED"
                }
            ]
        }
    },
    "details": []
} */
type DrainHostsResponse struct {
	Result DrainHostsResult `json:"results"`
}

// DrainHostsResult is the result contained in a DrainHostsResponse
type DrainHostsResult struct {
	ResultStatuses DrainHostsStatuses `json:"drainHostsResult"`
}

// DrainHostsStatuses contains list of individual []MaintenanceHostStatus
type DrainHostsStatuses struct {
	Status []MaintenanceHostStatus `json:"statuses"`
}

// EndMaintenanceResponse is returned from EndMaintenance()
/* {
    "responseCode": "OK",
    "serverInfo": {
        "clusterName": "dub-test",
        "thriftAPIVersion": 3,
        "statsUrlPrefix": ""
    },
    "result": {
        "endMaintenanceResult": {
            "statuses": [
                {
                    "host": "10.19.65.25",
                    "mode": "NONE"
                }
            ]
        }
    },
    "details": []
} */
type EndMaintenanceResponse struct {
	Result EndMaintenanceResult `json:"results"`
}

// EndMaintenanceResult is the result contained in a EndMaintenanceResponse
type EndMaintenanceResult struct {
	ResultStatuses EndMaintenanceStatuses `json:"endMaintenanceResult"`
}

// EndMaintenanceStatuses contains list of individual []MaintenanceHostStatus
type EndMaintenanceStatuses struct {
	Status []MaintenanceHostStatus `json:"statuses"`
}

/* TODO aurora maintenanceStatus / startMaintenance / endMaintenance / drainHosts requests all look like this:
{
	"hosts": {
		"hostNames": ["10.19.65.11"]
	}
}
*/

// MaintenanceRequest implements the payload for Aurora maintenance API calls
type MaintenanceRequest struct {
	MaintenanceHosts MaintenanceHostNames `json:"hosts"`
}

// MaintenanceHostNames implements the payload for each host in the Aurora MaintenanceRequest API cal
type MaintenanceHostNames struct {
	HostNames []string `json:"hostNames"`
}

// StartMaintenance puts nodes in maintenance mode via the Aurora API
func (c *Client) StartMaintenance(hosts map[string]string) error {
	url := fmt.Sprintf(c.APIUrl + "/startMaintenance")
	payload := genMaintenanceCallPayload(hosts)
	return auroraPostAPICall(url, payload)
}

// EndMaintenance takes node out of maintenance mode via the Aurora API
func (c *Client) EndMaintenance(hosts map[string]string) error {
	url := fmt.Sprintf(c.APIUrl + "/drainHosts")
	payload := genMaintenanceCallPayload(hosts)
	return auroraPostAPICall(url, payload)
}

// DrainHosts puts nodes into DRAINNG state via the Aurora API
func (c *Client) DrainHosts(hosts map[string]string) error {
	url := fmt.Sprintf(c.APIUrl + "/drainHosts")
	payload := genMaintenanceCallPayload(hosts)
	return auroraPostAPICall(url, payload)
}

func genMaintenanceCallPayload(hosts map[string]string) []byte {

	maintenanceHostNames := []string{}
	for host := range hosts {
		maintenanceHostNames = append(maintenanceHostNames, hosts[host])
	}

	maintenanceRequest := MaintenanceRequest{
		MaintenanceHosts: MaintenanceHostNames{maintenanceHostNames},
	}

	template, _ := json.Marshal(maintenanceRequest)
	return template
}

func auroraPostAPICall(url string, payload []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Print("Error preparing HTTP request: ", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Print("Error calling HTTP request: ", err)
		return err
	}

	defer resp.Body.Close()
	return nil
}

func getCurrentPath() string {

	gopath := os.Getenv("GOPATH")
	return filepath.Join(gopath, "src/github.com/alanbover/deathnode/aurora")
}
