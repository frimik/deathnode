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

// DrainHosts sets a list of mesos agents in DRAINING mode.
func (a *AuroraMonitor) DrainHosts(hosts map[string]string) error {

	drainHosts := make(map[string]string)
	for dnsName, ip := range hosts {
		if !a.IsDrained(ip) || !a.IsDraining(ip) {
			drainHosts[dnsName] = ip
		}
	}
	log.WithFields(log.Fields{
		"hosts": drainHosts,
	}).Info("Draining...")

	return a.ctx.AuroraConn.DrainHosts(drainHosts)
}

// StartMaintenance places list of mesos agents in MAINTENANCE mode.
func (a *AuroraMonitor) StartMaintenance(hosts map[string]string) error {

	maintenanceHosts := make(map[string]string)
	for dnsName, ip := range hosts {
		if !a.IsDrained(ip) || !a.IsDraining(ip) {
			maintenanceHosts[dnsName] = ip
		}
	}
	log.WithFields(log.Fields{
		"hosts": maintenanceHosts,
	}).Info("Starting Maintenance...")
	return a.ctx.AuroraConn.StartMaintenance(maintenanceHosts)
}

// EndMaintenance takes mesos agents out of (MAINTENANCE|DRAINING|DRAINED) modes
func (a *AuroraMonitor) EndMaintenance(hosts map[string]string) error {
	log.WithFields(log.Fields{
		"hosts": hosts,
	}).Info("Ending Maintenance...")
	return a.ctx.AuroraConn.EndMaintenance(hosts)
}

// IsDraining returns true if host is in DRAINING maintenance mode.
func (a *AuroraMonitor) IsDraining(host string) bool {

	host, ok := a.auroraCache.maintenance.Draining[host]
	if ok {
		log.Debugf("Host %s is DRAINING", host)
		return true
	}

	return false
}

// IsDrained returns true if host is in DRAINED maintenance mode.
func (a *AuroraMonitor) IsDrained(host string) bool {

	for _, h := range a.auroraCache.maintenance.Drained {
		if h == host {
			log.Debugf("Host %s is DRAINED", host)
			return true
		}
	}

	return false
}

// IsScheduled returns true if host is in SCHEDULED maintenance mode.
func (a *AuroraMonitor) isScheduled(host string) bool {

	for _, h := range a.auroraCache.maintenance.Scheduled {
		if h == host {
			log.Debugf("Host %s is SCHEDULED", host)
			return true
		}
	}

	return false
}
