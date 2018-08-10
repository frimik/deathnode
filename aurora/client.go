package aurora

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
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
	MasterURL string
	APIUrl    string // url for the /apibeta path
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
	ResultStatuses ResultStatuses `json:"maintenanceStatusResult"`
}

// ResultStatuses contains list of individual []HostStatus
type ResultStatuses struct {
	Status []HostStatus `json:"statuses"`
}

// HostStatus contains a Host and its Mode (Maintenance Mode) - one of NONE, SCHEDULED, DRAINING, DRAINED
type HostStatus struct {
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
	ResultStatuses ResultStatuses `json:"startMaintenanceResult"`
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
	ResultStatuses ResultStatuses `json:"drainHostsResult"`
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
	ResultStatuses ResultStatuses `json:"endMaintenanceResult"`
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
	Hosts HostNames `json:"hosts"`
}

// HostNames implements the payload for each host in the Aurora MaintenanceRequest API cal
type HostNames struct {
	HostNames []string `json:"hostNames"`
}

// StartMaintenance puts node in maintenance via the Aurora API
func (c *Client) StartMaintenance(hosts map[string]string) error {
	url := fmt.Sprintf(c.APIUrl + "/startMaintenance")
	payload := genStartMaintenanceCallPayload(hosts)
	return auroraPostAPICall(url, payload)
}

// UpdateMesosLeaderURL updates the URL to the currently leading Mesos Master
func (c *Client) UpdateMesosLeaderURL() (string, error) {
	u := fmt.Sprintf("%s/master/redirect", c.MasterURL)

	uParsed, err := url.Parse(u)
	if err != nil {
		log.WithField("error", err).Errorf("Unable to parse Master redirect URL: %s. Returning c.MasterURL", u)
		return c.MasterURL, nil
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(u)
	if err != nil {
		return "", err
	}

	leaderURL := resp.Header.Get("Location")
	log.Debugf("Mesos Master leaderURL is %s", leaderURL)

	lParsed, err := url.Parse(leaderURL)
	if err != nil {
		log.Fatal("Error parsing leaderURL")
	}

	if lParsed.Scheme == "" {
		lParsed.Scheme = uParsed.Scheme
	}

	c.LeaderURL = lParsed.String()
	log.Infof("Using leader: %s", c.LeaderURL)
	return c.LeaderURL, nil
}

// GetMesosTasks return the running tasks on the Mesos cluster
func (c *Client) GetMesosTasks() (*TasksResponse, error) {

	var tasks TasksResponse
	if err := c.getMesosTasksRecursive(&tasks, 0); err != nil {
		return nil, err
	}

	return &tasks, nil
}

func (c *Client) getMesosTasksRecursive(tasksResponse *TasksResponse, offset int) error {

	url := fmt.Sprintf("%s/master/tasks?limit=100&offset=%d", c.LeaderURL, offset)

	var tasks TasksResponse
	if err := mesosGetAPICall(url, &tasks); err != nil {
		return err
	}

	tasksResponse.Tasks = append(tasksResponse.Tasks, tasks.Tasks...)

	if len(tasks.Tasks) == 100 {
		c.getMesosTasksRecursive(tasksResponse, offset+100)
	}

	return nil
}

// GetMesosFrameworks returns the registered frameworks in Mesos
func (c *Client) GetMesosFrameworks() (*FrameworksResponse, error) {

	url := fmt.Sprintf(c.LeaderURL + "/master/state.json")

	var frameworks FrameworksResponse
	if err := mesosGetAPICall(url, &frameworks); err != nil {
		return nil, err
	}

	return &frameworks, nil
}

// GetMesosAgents returns the Mesos Agents registered in the Mesos cluster
func (c *Client) GetMesosAgents() (*SlavesResponse, error) {

	url := fmt.Sprintf(c.LeaderURL + "/master/slaves")

	var slaves SlavesResponse
	if err := mesosGetAPICall(url, &slaves); err != nil {
		return nil, err
	}

	return &slaves, nil
}

func genMaintenanceCallPayload(hosts map[string]string) []byte {

	maintenanceMachinesIDs := []MaintenanceMachinesID{}
	for host := range hosts {
		maintenanceMachinesID := MaintenanceMachinesID{
			Hostname: host,
			IP:       hosts[host],
		}
		maintenanceMachinesIDs = append(maintenanceMachinesIDs, maintenanceMachinesID)
	}

	maintenanceWindow := MaintenanceWindow{
		MachinesIds: maintenanceMachinesIDs,
		Unavailability: MaintenanceUnavailability{
			MaintenanceStart{
				Nanoseconds: 1,
			},
		},
	}

	maintenanceRequest := MaintenanceRequest{
		Windows: []MaintenanceWindow{maintenanceWindow},
	}

	template, _ := json.Marshal(maintenanceRequest)
	return template
}

func mesosGetAPICall(url string, response interface{}) error {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Print("Error preparing HTTP request: ", err)
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Print("Error calling HTTP request: ", err)
		return err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Print("Error decoding HTTP request: ", err)
		return err
	}

	return nil
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

func mesosPostAPICall(url string, payload []byte) error {

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
	return filepath.Join(gopath, "src/github.com/alanbover/deathnode/mesos")
}
