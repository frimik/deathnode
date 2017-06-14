package deathnode

// Given an autoscaling group, decides which is/are the best agent/s to kill

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/mesos"
	log "github.com/sirupsen/logrus"
)

type DeathNodeWatcher struct {
	notebook     *Notebook
	mesosMonitor *mesos.MesosMonitor
	constraints  constraint
	recommender  recommender
}

func NewDeathNodeWatcher(notebook *Notebook, mesosMonitor *mesos.MesosMonitor, constraintType, recommenderType string) *DeathNodeWatcher {

	contrainsts, err := newConstraint(constraintType)
	if err != nil {
		log.Fatal(err)
	}

	recommender, err := newRecommender(recommenderType)
	if err != nil {
		log.Fatal(err)
	}

	return &DeathNodeWatcher{
		notebook:     notebook,
		mesosMonitor: mesosMonitor,
		constraints:  contrainsts,
		recommender:  recommender,
	}
}

func (y *DeathNodeWatcher) RemoveUndesiredInstances(autoscalingMonitor *aws.AutoscalingGroupMonitor) error {

	numUndesiredInstances := autoscalingMonitor.NumUndesiredInstances()
	log.Debugf("Undesired Mesos Agents: %d", numUndesiredInstances)

	removedInstances := 0

	for removedInstances < numUndesiredInstances {
		allowedInstancesToKill := y.constraints.filter(autoscalingMonitor.GetInstancesNotMarkedToBeRemoved())
		bestInstanceToKill := y.recommender.find(allowedInstancesToKill)
		log.Debugf("Mark instance %s for removal", bestInstanceToKill.GetInstanceId())
		err := bestInstanceToKill.MarkToBeRemoved()
		if err != nil {
			log.Errorf("Unable to mark instance %s for removal", bestInstanceToKill.GetIP())
			log.Error(err)
			break
		}

		removedInstances += 1
	}

	return nil
}

func (y *DeathNodeWatcher) DestroyInstancesAttempt() {

	err := y.notebook.DestroyInstancesAttempt()
	if err != nil {
		log.Error(err)
	}
}
