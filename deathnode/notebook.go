package deathnode

// Stores the Mesos agents we want to kill. It will periodically review the state of the agents and kill them if
// they are not running any tasks

import (
	"time"

	"github.com/alanbover/deathnode/context"
	"github.com/alanbover/deathnode/monitor"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
)

// Notebook stores the necessary information for deal with instances that should be deleted
type Notebook struct {
	mesosMonitor        *monitor.MesosMonitor
	auroraMonitor       *monitor.AuroraMonitor
	autoscalingGroups   *monitor.AutoscalingServiceMonitor
	lastDeleteTimestamp time.Time
	ctx                 *context.ApplicationContext
}

// NewNotebook creates a notebook object, which is in charge of monitoring and delete instances marked to be deleted
func NewNotebook(ctx *context.ApplicationContext, autoscalingGroups *monitor.AutoscalingServiceMonitor,
	mesosMonitor *monitor.MesosMonitor, auroraMonitor *monitor.AuroraMonitor) *Notebook {

	return &Notebook{
		mesosMonitor:        mesosMonitor,
		auroraMonitor:       auroraMonitor,
		autoscalingGroups:   autoscalingGroups,
		lastDeleteTimestamp: time.Time{},
		ctx:                 ctx,
	}
}

func (n *Notebook) setAgentsInMaintenance(instances []*ec2.Instance) error {

	hosts := map[string]string{}
	for _, instance := range instances {
		hosts[*instance.PrivateDnsName] = *instance.PrivateIpAddress
	}

	log.WithFields(log.Fields{
		"hosts": hosts,
	}).Info("Starting Mesos agent maintenance")

	return n.mesosMonitor.SetMesosAgentsInMaintenance(hosts)
}

func (n *Notebook) drainAgent(instance *ec2.Instance) error {
	hosts := map[string]string{}
	hosts[*instance.PrivateDnsName] = *instance.PrivateIpAddress

	log.WithFields(log.Fields{
		"instance_id": *instance.InstanceId,
		"ip":          *instance.PrivateIpAddress,
	}).Info("Draining Mesos agent")
	return n.auroraMonitor.DrainHosts(hosts)
}

func (n *Notebook) endMaintenance(instanceMonitor *monitor.InstanceMonitor) error {

	hosts := map[string]string{}
	hosts[instanceMonitor.IP()] = instanceMonitor.IP()
	log.WithFields(log.Fields{
		"instance_id": *instanceMonitor.InstanceID(),
		"ip":          instanceMonitor.IP(),
	}).Info("Ending Mesos agent maintenance")
	return n.ctx.AuroraConn.EndMaintenance(hosts)
}

func (n *Notebook) shouldWaitForNextDestroy() bool {
	return n.ctx.Clock.Since(n.lastDeleteTimestamp).Seconds() <= float64(n.ctx.Conf.DelayDeleteSeconds)
}

func (n *Notebook) destroyInstance(instanceMonitor *monitor.InstanceMonitor) error {

	if instanceMonitor.LifecycleState() == monitor.LifecycleStateTerminatingWait {
		defer n.endMaintenance(instanceMonitor)

		log.Infof("Destroy instance %s", *instanceMonitor.InstanceID())
		err := n.ctx.AwsConn.CompleteLifecycleAction(
			instanceMonitor.AutoscalingGroupID(), instanceMonitor.InstanceID())
		if err != nil {
			log.Errorf("Unable to complete lifecycle action on instance %s", *instanceMonitor.InstanceID())
			return err
		}
		if n.ctx.Conf.DelayDeleteSeconds != 0 {
			n.lastDeleteTimestamp = n.ctx.Clock.Now()
		}
	} else {
		log.Debugf("Instance %s waiting for AWS to start termination lifecycle", *instanceMonitor.InstanceID())
	}
	return nil
}

func (n *Notebook) resetLifecycle(instanceMonitor *monitor.InstanceMonitor) {

	// Check if timeout is close to expire
	startTimeoutTimestamp := time.Unix(instanceMonitor.TagRemovalTimestamp(), 0)
	maxSecondsToRefresh := float64(n.ctx.Conf.LifecycleTimeout) * monitor.LifeCycleRefreshTimeoutPercentage

	if instanceMonitor.LifecycleState() == monitor.LifecycleStateTerminatingWait && n.ctx.Clock.Since(startTimeoutTimestamp).Seconds() > maxSecondsToRefresh {
		err := instanceMonitor.RefreshLifecycleHook()
		if err != nil {
			log.Errorf("Unable to reset lifecycle hook for instance %s", *instanceMonitor.InstanceID())
		}
	}
}

func (n *Notebook) destroyInstanceAttempt(instance *ec2.Instance) error {

	log.Debugf("Starting process to delete instance %s", *instance.InstanceId)

	instanceMonitor, err := n.autoscalingGroups.GetInstanceByID(*instance.InstanceId)
	if err != nil {
		return err
	}

	// If the instance is protected, remove instance protection
	n.removeInstanceProtection(instanceMonitor)

	// Reset lifecycle hook timeout if needed
	if n.ctx.Conf.ResetLifecycle {
		n.resetLifecycle(instanceMonitor)
	}

	// Check if we need to wait before destroy another instance
	if n.shouldWaitForNextDestroy() {
		log.Debugf("Seconds since last destroy: %v. Instance %s will not be destroyed",
			n.ctx.Clock.Since(n.lastDeleteTimestamp).Seconds(), *instance.InstanceId)
		return nil
	}

	// If we're using Aurora - Ensure instance is DRAINING
	if n.ctx.Conf.AuroraURL != "" {
		if err := n.drainAgent(instance); err != nil {
			return err
		}
	}

	// If the instance can be killed, delete it
	if !n.mesosMonitor.IsProtected(*instance.PrivateIpAddress) {
		if err := n.destroyInstance(instanceMonitor); err != nil {
			return err
		}
	}
	return nil
}

// DestroyInstancesAttempt iterates around all instances marked to be deleted, and:
// - set them in maintenance
// - remove instance protection
// - complete lifecycle action if there is no tasks running from the protected frameworks
func (n *Notebook) DestroyInstancesAttempt() error {

	// Get instances marked for removal
	instances, err := n.ctx.AwsConn.DescribeInstancesByTag(n.ctx.Conf.DeathNodeMark)
	if err != nil {
		log.Debugf("Error retrieving instances with tag %s", n.ctx.Conf.DeathNodeMark)
		return err
	}

	if len(instances) > 0 {
		// Set instances in maintenance
		n.setAgentsInMaintenance(instances)

		for _, instance := range instances {
			if err := n.destroyInstanceAttempt(instance); err != nil {
				log.Warn(err)
			}
		}
	}

	return nil
}

func (n *Notebook) removeInstanceProtection(instance *monitor.InstanceMonitor) error {

	if instance.IsProtected() {
		return instance.RemoveInstanceProtection()
	}

	return nil
}
