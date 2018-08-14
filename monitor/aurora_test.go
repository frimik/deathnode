package monitor

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/alanbover/deathnode/aurora"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAuroraSetMaintenance(t *testing.T) {
	Convey("When generating the payload for a maintenance call", t, func() {
		auroraConn := &aurora.ClientMock{
			Records: map[string]*[]string{},
		}
		templateJSON := aurora.MaintenanceRequest{}
		var testValues = []struct {
			hosts map[string]string
			num   int
		}{
			{map[string]string{}, 0},
			{map[string]string{"hostname1": "10.0.0.1"}, 1},
			{map[string]string{"hostname1": "10.0.0.1", "hostname2": "10.0.0.2"}, 2},
		}

		for _, testValue := range testValues {
			Convey(fmt.Sprintf("it should be possible to start maintenance for %v agents", testValue.num), func() {
				template := auroraConn.GenMaintenanceCallPayload(testValue.hosts)
				json.Unmarshal(template, &templateJSON)
				So(len(templateJSON.MaintenanceHosts.HostNames), ShouldEqual, testValue.num)
			})
		}
	})
}
