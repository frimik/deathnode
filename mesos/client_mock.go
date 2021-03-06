package mesos

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// ClientMock implements mesos.ClientInterface for testing purposes
type ClientMock struct {
	Records  map[string]*[]string
	Requests map[string]*[]string
}

// GetMesosTasks mocked for testing purposes
func (c *ClientMock) GetMesosTasks() (*TasksResponse, error) {
	mockResponse, _ := c.replay(&TasksResponse{}, "GetMesosTasks")
	return mockResponse.(*TasksResponse), nil
}

// GetMesosFrameworks mocked for testing purposes
func (c *ClientMock) GetMesosFrameworks() (*FrameworksResponse, error) {
	mockResponse, _ := c.replay(&FrameworksResponse{}, "GetMesosFrameworks")
	return mockResponse.(*FrameworksResponse), nil
}

// UpdateMesosLeaderURL mocked for testing purposes
func (c *ClientMock) UpdateMesosLeaderURL() (string, error) {
	return "http://leader.mesos", nil
}

// GetMesosAgents mocked for testing purposes
func (c *ClientMock) GetMesosAgents() (*SlavesResponse, error) {
	mockResponse, _ := c.replay(&SlavesResponse{}, "GetMesosSlaves")
	return mockResponse.(*SlavesResponse), nil
}

// GenMaintenanceCallPayload mocked for testing purposes
func (c *ClientMock) GenMaintenanceCallPayload(hosts map[string]string) []byte {
	return genMaintenanceCallPayload(hosts)
}

// SetHostsInMaintenance mocked for testing purposes
func (c *ClientMock) SetHostsInMaintenance(hosts map[string]string) error {
	if c.Requests == nil {
		c.Requests = map[string]*[]string{}
	}

	hostsCallArguments := []string{}
	for _, host := range hosts {
		hostsCallArguments = append(hostsCallArguments, host)
		hostsCallArguments = append(hostsCallArguments, hosts[host])
	}

	c.Requests["SetHostInMaintenance"] = &hostsCallArguments
	return nil
}

func (c *ClientMock) replay(mockResponse interface{}, templateFileName string) (interface{}, error) {

	records, ok := c.Records[templateFileName]
	if !ok {
		fmt.Printf("AWS Mock %v method called but not defined\n", templateFileName)
		os.Exit(1)
	}

	if len(*records) == 0 {
		fmt.Printf("AWS Mock replay called more times than configured for %v\n", templateFileName)
		os.Exit(1)
	}

	currentRecord := (*records)[0]

	file, err := ioutil.ReadFile(getCurrentPath() + "/testdata" + "/" + currentRecord + "/" + templateFileName + ".json")
	if err != nil {
		fmt.Printf("File error: %v\n", err)
		os.Exit(1)
	}

	err = json.Unmarshal(file, &mockResponse)
	if err != nil {
		fmt.Printf("Error loading mock json: %v\n", err)
		os.Exit(1)
	}

	*records = (*records)[1:]
	return mockResponse, nil
}
