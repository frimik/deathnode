package monitor

// Monitor holds a connection to mesos, and a cache for every iteration
// With MesosCache we reduce the number of calls to mesos, also we map it for quicker access

import (
	"github.com/alanbover/deathnode/aurora"
	"github.com/alanbover/deathnode/context"
	log "github.com/sirupsen/logrus"
)

// AuroraMonitor monitors the aurora cluster, creating a cache to reduce the number of calls against it
type AuroraMonitor struct {
	auroraCache *auroraCache
	ctx         *context.ApplicationContext
}

// AuroraCache stores the objects of the auroraApi in a way that is directly accesible
// tasks: map[slaveId][]Task
// frameworks: map[frameworkID]Framework
// slaves: map[privateIPAddress]Slave
type auroraCache struct {
	maintenance aurora.MaintenanceResponse
}

// NewAuroraMonitor returns a new aurora.monitor object
func NewAuroraMonitor(ctx *context.ApplicationContext) *AuroraMonitor {

	return &AuroraMonitor{
		auroraCache: &auroraCache{
			maintenance: aurora.MaintenanceResponse{},
		},
		ctx: ctx,
	}
}

// Refresh updates the aurora cache
func (a *AuroraMonitor) Refresh() {

	a.auroraCache.maintenance = a.getMaintenance()
}

func (a *AuroraMonitor) getMaintenance() aurora.MaintenanceResponse {

	maintenanceResponse, err := a.ctx.AuroraConn.GetMaintenance()
	if err != nil {
		log.Warning(err)
		return *maintenanceResponse
	}

	return *maintenanceResponse

}

// DrainHosts sets a list of mesos agents in DRAINING mode. Aurora only.
func (a *AuroraMonitor) DrainHosts(hosts map[string]string) error {
	log.WithFields(log.Fields{
		"hosts": hosts,
	}).Info("Draining...")

	return a.ctx.AuroraConn.DrainHosts(hosts)
}
