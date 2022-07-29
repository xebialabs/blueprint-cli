package k8s

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/xebialabs/blueprint-cli/pkg/util"
)

type Command struct {
	Maincmd string
	Args    []string
}

type Kube struct {
	Namespace string
	Command
}

type Resource struct {
	Kube
	Type string
	Name ResourceName
	spin *spinner.Spinner
}

type ResourceName interface{}

func (c Command) Run() (string, bool) {
	output, err := util.ProcessCmdResult(*exec.Command(c.Maincmd, c.Args...))
	if err != nil {
		return string(output), false
	} else {
		return string(output), true
	}
}

func (r Resource) CreateResource(namespace, resourceType string, resourceName ResourceName) Resource {
	resource := Resource{
		Kube: Kube{
			Namespace: namespace,
			Command: Command{
				Maincmd: "kubectl",
			},
		},
		Type: resourceType,
		Name: resourceName,
		spin: spinner.New(spinner.CharSets[9], 100*time.Millisecond),
	}
	return resource
}

func (r Resource) DeleteResource(pattern string) {
	r.DeleteFilteredResources([]string{pattern}, true, false)
}

func (r Resource) DeleteResourceStartsWith(pattern string) {
	r.DeleteFilteredResources([]string{pattern}, false, false)
}

func (r Resource) DeleteFilteredResources(patterns []string, anyPosition, force bool) {
	r.spin.Start()
	if name, status := r.Name.(string); status {
		if force {
			r.Args = []string{"delete", r.Type, name, "-n", r.Namespace, "--force"}
		} else {
			r.Args = []string{"delete", r.Type, name, "-n", r.Namespace}
		}
		r.spin.Prefix = fmt.Sprintf("Deleting %s...\t", name)
		if output, ok := r.Run(); ok {
			output = strings.Replace(output, "\n", "", -1)
			r.spin.Prefix = output + "\t"
		} else {
			util.Fatal("\nError while deleting %s\n", name)
		}
	} else {
		// Delete logic by pattern matching
		tokens := r.GetResources()

		for _, value := range tokens {
			found := true
			for _, pattern := range patterns {
				hasPattern := false
				if anyPosition {
					hasPattern = strings.Contains(value, pattern)
				} else {
					hasPattern = strings.HasPrefix(value, pattern)
				}
				if !(found && hasPattern && !strings.Contains(value, "/")) {
					found = false
					break
				}
			}
			if found {
				r.Args = []string{"delete", r.Type, value, "-n", r.Namespace}
				output, ok := r.Run()
				if !ok {
					util.Fatal("Error while deleting %s: %s\n", r.Type, value)
				} else {
					util.Info(output)
				}
			}
		}
	}
	r.spin.Stop()
}

func (r Resource) RemoveFinalizers(pattern string) {
	r.spin.Start()
	if name, status := r.Name.(string); status {
		r.Args = []string{"patch", r.Type, name, "-n", r.Namespace, "-p", "{\"metadata\":{\"finalizers\":[]}}", "--type=merge"}
		r.spin.Prefix = fmt.Sprintf("Deleting %s...\t", name)
		if output, ok := r.Run(); ok {
			output = strings.Replace(output, "\n", "", -1)
			r.spin.Prefix = output + "\t"
		} else {
			util.Fatal("\nError while deleting %s\n", name)
		}
	} else {
		// Delete logic by pattern matching
		tokens := r.GetResources()

		for _, value := range tokens {
			if strings.Contains(value, pattern) && !strings.Contains(value, "/") {
				r.Args = []string{"delete", r.Type, value, "-n", r.Namespace}
				output, ok := r.Run()
				if !ok {
					util.Fatal("Error while deleting %s: %s\n", r.Type, value)
				} else {
					util.Info(output)
				}
			}
		}
	}
	r.spin.Stop()
}

func (r Resource) GetFilteredResource(pattern string) string {
	r.spin.Start()
	r.spin.Prefix = fmt.Sprintf("Fetching %s from %s namespace\t", r.Type, r.Namespace)
	r.Command.Args = []string{"get", r.Type, "-n", r.Namespace, "-o", "custom-columns=:metadata.name", "--sort-by=metadata.name"}
	output, ok := r.Command.Run()
	if ok {
		r.spin.Prefix = fmt.Sprintf("Resources of type %s fetched successfully\n\t", r.Type)
	} else {
		util.Fatal("Error occurred while fetching resource of type %s\n", r.Type)
	}

	output = strings.Replace(output, "\n", " ", -1)
	tokens := strings.TrimSpace(strings.Split(output, " "))

	for _, value := range tokens {
		if strings.Contains(value, pattern) && !strings.Contains(value, "/") {
			r.spin.Stop()
			return value
		}
	}

	r.spin.Stop()
	return ""
}

func (r Resource) GetResources() []string {
	r.spin.Start()
	r.spin.Prefix = fmt.Sprintf("Fetching %s from %s namespace\t", r.Type, r.Namespace)
	r.Command.Args = []string{"get", r.Type, "-n", r.Namespace, "-o", "custom-columns=:metadata.name", "--sort-by=metadata.name"}
	output, ok := r.Command.Run()
	if ok {
		r.spin.Prefix = fmt.Sprintf("Resources of type %s fetched successfully\n\t", r.Type)
	} else {
		util.Fatal("Error occurred while fetching resource of type %s\n", r.Type)
	}

	output = strings.Replace(output, "\n", " ", -1)
	tokens := strings.TrimSpace(strings.Split(output, " "))

	r.spin.Stop()
	return tokens
}
