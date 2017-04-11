package aws

import (
	"testing"
)

func TestNewAutoscalingGroupMonitor(t *testing.T) {

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName":     &[]string{"default"},
		},
	}

	monitor, _ := NewAutoscalingGroupMonitor(awsConn, "some-Autoscaling-Group")

	if monitor == nil {
		t.Fatal("TestNewAutoscalingGroupMonitor return nil")
	}

	if (len(monitor.autoscaling.instanceMonitors)) != 3 {
		t.Fatal("Incorrect number of instances in ASG")
	}

	if monitor.NumUndesiredInstances() > 0 {
		t.Fatal("Incorrect number of undesired ASG instances")
	}
}

func TestNumUndesiredASGinstances(t *testing.T) {

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName":     &[]string{"one_undesired_host"},
		},
	}

	monitor, _ := NewAutoscalingGroupMonitor(awsConn, "some-Autoscaling-Group")

	if monitor.NumUndesiredInstances() != 1 {
		t.Fatal("Incorrect number of undesired ASG instances")
	}
}

func TestNewAutoscalingGroups(t *testing.T) {

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName":     &[]string{"default"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)

	if (*autoscalingGroups)[0].autoscaling.autoscalingGroupName != "some-Autoscaling-Group" {
		t.Fatal("Error creating AutoscalingGroups")
	}
}

func TestRemoveInstanceFromAutoscalingGroup(t *testing.T) {

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName":     &[]string{"default"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)

	instanceMonitors := (*autoscalingGroups)[0].autoscaling.instanceMonitors
	instanceMonitors["i-34719eb8"].RemoveFromAutoscalingGroup()

	callArguments := awsConn.Requests["DetachInstance"]

	if (*callArguments)[0] != "some-Autoscaling-Group" || (*callArguments)[1] != "i-34719eb8" {
		t.Fatal("Incorrect parameters when calling RemoveFromAutoscalingGroup")
	}
}

func TestRefresh(t *testing.T) {

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default", "default", "default", "default"},
			"DescribeAGByName":     &[]string{"default", "refresh"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)

	autoscalingGroup := (*autoscalingGroups)[0]
	autoscalingGroup.Refresh()

	if len(autoscalingGroup.autoscaling.instanceMonitors) != 3 {
		t.Fatal("Incorrect number of elements after refresh()")
	}

	expectedInstanceIds := []string{"i-34719eb8", "i-777a73cf", "i-666ca923"}

	for _, instanceId := range expectedInstanceIds {
		_, ok := autoscalingGroup.autoscaling.instanceMonitors[instanceId]
		if !ok {
			t.Fatal("Incorrect instanceId found after refresh()")
		}
	}

}