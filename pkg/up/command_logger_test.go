package up

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type LogSpy struct {
	Val chan string
}

func (spy *LogSpy) Callback(currentStage string) {
	spy.Val <- currentStage
}

func Test_logCapture(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []string
	}{
		{
			"get generatedPlan output from logs for single deploy item",
			[]byte(`2020-01-28 14:21:27.539 [main] {} INFO  c.x.d.s.deployment.DeploymentService - Generated plan for task dac34110-6c10-45dc-aa94-a70bee010f97:
			    # [Phased plan]
			    # [Plan phase] Deploy
			     * Deploy K8s-NameSpace 1.0 on K8S
			       -------------------------------
			`),
			[]string{"Deploying K8s-NameSpace\n\n"},
		},
		{
			"get executedLog output from logs for single deploy item",
			[]byte(`2020-01-28 14:24:34.970 [task-sys-xl.task.dispatchers.state-management-dispatcher-16] {taskId=d9172126-e3f8-4b77-94ae-94b9558fd4c1, username=admin} INFO  c.x.d.e.tasker.TaskManagingActor - Task [d9172126-e3f8-4b77-94ae-94b9558fd4c1] is completed with state [DONE]
			`),
			[]string{"Deployed K8s-NameSpace\n\n"},
		},
		{
			"get failExecutedLog output from logs for single deploy item",
			[]byte(`2020-01-28 14:23:24.506 [task-sys-xl.task.dispatchers.state-management-dispatcher-51] {taskId=3d1a86d3-bd9b-40bc-9043-50ac4a7894f4, username=admin} INFO  c.x.d.e.tasker.TaskManagingActor - Task [3d1a86d3-bd9b-40bc-9043-50ac4a7894f4] is completed with state [FAILED]
			`),
			[]string{
				"Failed deployment for K8s-NameSpace\n\n",
				"Undeploying K8s-NameSpace\n\n",
			},
		},
		{
			"get generatedPlan output from logs for multiple deploy item",
			[]byte(`2020-01-28 14:21:43.870 [main] {} INFO  c.x.d.s.deployment.DeploymentService - Generated plan for task c97248b4-06a7-41d0-816d-3fdc84b2f442:
            # [Phased plan]
            # [Plan phase] Deploy
            #####################################################################################################################################
            # [Serial] Deploy K8s-Ingress-Controller v0.6, PostgreSQL 10.5, Answers-Configmap-Deployment 21ea1ce3f6892fc6b82693a515a46251 on K8S
            #####################################################################################################################################
			`),
			[]string{"Deploying K8s-Ingress-Controller\n\nDeploying PostgreSQL\n\nDeploying Answers-Configmap-Deployment\n\n"},
		},
		{
			"get executedLog output from logs for multiple deploy item",
			[]byte(`2020-01-28 14:23:24.506 [task-sys-xl.task.dispatchers.state-management-dispatcher-51] {taskId=3d1a86d3-bd9b-40bc-9043-50ac4a7894f4, username=admin} INFO  c.x.d.e.tasker.TaskManagingActor - Task [3d1a86d3-bd9b-40bc-9043-50ac4a7894f4] is completed with state [DONE]
			`),
			[]string{"Deployed K8s-Ingress-Controller\n\nDeployed PostgreSQL\n\nDeployed Answers-Configmap-Deployment\n\n"},
		},
		{
			"get failExecutedLog output from logs for multiple deploy item",
			[]byte(`2020-01-28 14:23:24.506 [task-sys-xl.task.dispatchers.state-management-dispatcher-51] {taskId=3d1a86d3-bd9b-40bc-9043-50ac4a7894f4, username=admin} INFO  c.x.d.e.tasker.TaskManagingActor - Task [3d1a86d3-bd9b-40bc-9043-50ac4a7894f4] is completed with state [FAILED]
			`),
			[]string{
				"Failed deployment for K8s-Ingress-Controller\n\nFailed deployment for PostgreSQL\n\nFailed deployment for Answers-Configmap-Deployment\n\n",
				"Undeploying K8s-Ingress-Controller\n\nUndeploying PostgreSQL\n\nUndeploying Answers-Configmap-Deployment\n\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spy := &LogSpy{make(chan string)}
			// This routine is required for channels to work
			go func() {
				lastWritten = ""
				logCapture(tt.data, spy.Callback)
			}()
			for _, exp := range tt.expected {
				select {
				case out := <-spy.Val:
					assert.Equal(t, exp, out)
				case <-time.After(10 * time.Second):
					t.Errorf("Timed out")
				}
			}
		})
	}
}

func Test_getCurrentTask(t *testing.T) {
	tests := []struct {
		name     string
		eventLog string
		want     string
	}{
		{
			"empty when no task id",
			"2020-01-28 14:21:27.539 [main] {} INFO  c.x.d.s.deployment.DeploymentService\n",
			"",
		},
		{
			"get task id from log string",
			"2020-01-28 14:21:27.539 [main] {} INFO  c.x.d.s.deployment.DeploymentService - Generated plan for task dac34110-6c10-45dc-aa94-a70bee010f97:\n",
			"dac34110-6c10-45dc-aa94-a70bee010f97:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCurrentTask(tt.eventLog, strings.Index(tt.eventLog, generatedPlan))

			assert.Equal(t, tt.want, got)
		})
	}
}
