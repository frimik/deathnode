package mesos_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/alanbover/deathnode/mesos"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSetMesosAgentsInMaintenance(t *testing.T) {
	Convey("When generating the payload for a maintenance call", t, func() {
		mesosConn := &mesos.ClientMock{
			Records: map[string]*[]string{},
		}
		templateJSON := mesos.MaintenanceRequest{}
		var testValues = []struct {
			hosts map[string]string
			num   int
		}{
			{map[string]string{}, 0},
			{map[string]string{"hostname1": "10.0.0.1"}, 1},
			{map[string]string{"hostname1": "10.0.0.1", "hostname2": "10.0.0.2"}, 2},
		}

		for _, testValue := range testValues {
			Convey(fmt.Sprintf("it should be possible to configure for %v agents", testValue.num), func() {
				template := mesosConn.GenMaintenanceCallPayload(testValue.hosts)
				json.Unmarshal(template, &templateJSON)
				So(len(templateJSON.Windows[0].MachinesIds), ShouldEqual, testValue.num)
			})
		}
	})
}
