package k8s

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/xebialabs/blueprint-cli/pkg/osHelper"
	"github.com/xebialabs/blueprint-cli/pkg/util"
)

type Command struct {
	Maincmd string
	Args    []string
	StdOut  *os.File
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
	if c.StdOut == nil {
		output, err := util.ProcessCmdResult(*exec.Command(c.Maincmd, c.Args...))
		if err != nil {
			return string(output), false
		} else {
			return string(output), true
		}
	} else {
		cmd := exec.Command(c.Maincmd, c.Args...)
		cmd.Stdout = c.StdOut
		err := util.ProcessCmdResultWithoutOutput(*cmd)
		if err != nil {
			return err.Error(), false
		} else {
			return "", true
		}
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

type confirmFn func(string, string) (bool, error)

func (r Resource) DeleteResource(pattern string, confirm confirmFn, backupPath string) {
	r.DeleteFilteredResources([]string{pattern}, true, false, confirm, backupPath)
}

func (r Resource) DeleteResourceStartsWith(pattern string, confirm confirmFn, backupPath string) {
	r.DeleteFilteredResources([]string{pattern}, false, false, confirm, backupPath)
}

func (r Resource) DeleteFilteredResources(patterns []string, anyPosition, force bool, confirm confirmFn, backupPath string) {
	if name, status := r.Name.(string); status && name != "" {
		if force {
			r.Args = []string{"delete", r.Type, name, "-n", r.Namespace, "--force"}
		} else {
			r.Args = []string{"delete", r.Type, name, "-n", r.Namespace}
		}
		r.spin.Stop()
		if doDelete, err := confirm(r.Type, name); doDelete && err == nil {
			r.spin.Prefix = fmt.Sprintf("Deleting %s/%s...\t", r.Type, name)
			r.spin.Start()
			defer r.spin.Stop()

			if backupPath != "" {
				filepath := r.Filename(".yaml")
				if err = r.SaveYamlFile(filepath); err != nil {
					util.Fatal("Error while deleting %s\n", r.ResourceName())
				}
			}

			if output, ok := r.Run(); ok {
				output = strings.Replace(output, "\n", "", -1)
				util.Verbose(output + "\n")
			} else {
				util.Fatal("\nError while deleting %s\n", r.ResourceName())
			}

		} else if err != nil {
			util.Fatal("Error while deleting %s: %s\n", r.ResourceName(), err)
		} else {
			util.Info("Skipping delete of the resource %s\n", r.ResourceName())
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
			r.spin.Stop()
			if found {
				if doDelete, err := confirm(r.Type, value); doDelete && err == nil {
					r.spin.Prefix = fmt.Sprintf("Deleting %s/%s...\t", r.Type, value)
					r.spin.Start()
					defer r.spin.Stop()

					if backupPath != "" {
						filepath := r.filename(value, ".yaml")
						if err = r.saveYamlFile(value, filepath); err != nil {
							util.Fatal("Error while deleting %s/%s\n", r.Type, value)
						}
					}

					r.Args = []string{"delete", r.Type, value, "-n", r.Namespace}
					output, ok := r.Run()
					if !ok {
						util.Fatal("Error while deleting %s/%s\n", r.Type, value)
					} else {
						util.Info(output + "\n")
					}
				} else if err != nil {
					util.Fatal("Error while deleting %s/%s: %s\n", r.Type, value, err)
				} else {
					util.Info("Skipping delete of the resource %s/%s", r.Type, value)
				}
			}
		}
	}
	r.spin.Stop()
}

func (r Resource) RemoveFinalizers(pattern string) {
	r.spin.Start()
	defer r.spin.Stop()

	if name, status := r.Name.(string); status && name != "" {
		r.Args = []string{"patch", r.Type, name, "-n", r.Namespace, "-p", "{\"metadata\":{\"finalizers\":[]}}", "--type=merge"}
		r.spin.Prefix = fmt.Sprintf("Deleting finalizers %s/%s...\t", r.Type, name)
		if output, ok := r.Run(); ok {
			output = strings.Replace(output, "\n", "", -1)
			util.Info(output + "\n")
		} else {
			util.Fatal("\nError while deleting %s/%s\n", r.Type, name)
		}
	} else {
		// Delete logic by pattern matching
		tokens := r.GetResources()

		for _, value := range tokens {
			if strings.Contains(value, pattern) && !strings.Contains(value, "/") {
				r.Args = []string{"delete", r.Type, value, "-n", r.Namespace}
				r.spin.Prefix = fmt.Sprintf("Deleting finalizers %s/%s...\t", r.Type, value)
				output, ok := r.Run()
				if !ok {
					util.Fatal("Error while deleting %s/%s\n", r.Type, value)
				} else {
					util.Info(output + "\n")
				}
			}
		}
	}
}

func (r Resource) GetFilteredResource(patterns []string, anyPosition bool) string {

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
			util.Verbose("GetFilteredResource returning %s\n", value)
			return value
		}
	}

	return ""
}

func (r Resource) GetFilteredResources(patterns []string, anyPosition bool) []string {
	filtered := []string{}
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
			filtered = append(filtered, value)
		}
	}

	util.Verbose("GetFilteredResources returning %s\n", strings.Join(filtered, ","))
	return filtered
}

func (r Resource) GetResources() []string {
	return r.GetResourcesWithCustomAttrs("--sort-by=metadata.name", "--ignore-not-found=true")
}

func (r Resource) GetResourcesWithCustomAttrs(appendedAttrs ...string) []string {
	r.spin.Start()
	defer r.spin.Stop()

	r.spin.Prefix = fmt.Sprintf("Fetching %s from %s namespace...\t", r.Type, r.Namespace)
	r.Command.Args = append([]string{"get", r.Type, "-n", r.Namespace, "-o", "custom-columns=:metadata.name"}, appendedAttrs...)
	if name, status := r.Name.(string); status && name != "" {
		r.Command.Args = append([]string{"get", r.Type, name, "-n", r.Namespace, "-o", "custom-columns=:metadata.name"}, appendedAttrs...)
	}
	output, ok := r.Command.Run()
	if ok {
		util.Info("Resources of type %s fetched successfully\n", r.Type)
	} else {
		util.Fatal("Error occurred while fetching resource of type %s\n", r.Type)
	}

	util.Verbose("GetResources output: %s\n", output)

	output = strings.TrimSpace(strings.Replace(output, "\n", " ", -1))
	tokens := strings.Split(output, " ")

	filtered := []string{}
	for _, value := range tokens {
		if len(strings.TrimSpace(value)) > 0 {
			filtered = append(filtered, value)
		}
	}

	return filtered
}

func (r Resource) ExistResource() bool {
	return len(r.GetResources()) > 0
}

func (r Resource) Status() string {
	r.spin.Start()
	defer r.spin.Stop()

	r.spin.Prefix = fmt.Sprintf("Fetching status %s from %s namespace...\t", r.Type, r.Namespace)

	r.Command.Args = []string{"get", r.Type, "-n", r.Namespace, "--no-headers", "-o", "custom-columns=:status.phase"}
	if name, status := r.Name.(string); status && name != "" {
		r.Command.Args = []string{"get", r.Type, name, "-n", r.Namespace, "--no-headers", "-o", "custom-columns=:status.phase"}
	}
	output, ok := r.Command.Run()
	if ok {
		util.Info("Resources of type %s fetched status successfully\n", r.Type)
	} else {
		util.Fatal("Error occurred while fetching resource status of type %s\n", r.Type)
	}

	util.Verbose("Get status output: %s\n", output)

	output = strings.TrimSpace(strings.Replace(output, "\n", " ", -1))

	return output
}

func (r Resource) StatusReason() string {
	r.spin.Start()
	r.spin.Prefix = fmt.Sprintf("Fetching status reason %s from %s namespace...\t", r.Type, r.Namespace)
	r.Command.Args = []string{"get", r.Type, "-n", r.Namespace, "--no-headers", "-o", "custom-columns=:status.reason"}
	if name, status := r.Name.(string); status && name != "" {
		r.Command.Args = []string{"get", r.Type, name, "-n", r.Namespace, "--no-headers", "-o", "custom-columns=:status.reason"}
	}
	output, ok := r.Command.Run()
	if ok {
		util.Info("Resources of type %s fetched status reason successfully\n", r.Type)
	} else {
		util.Fatal("Error occurred while fetching resource status reason of type %s\n", r.Type)
	}

	util.Verbose("Get status reason output: %s\n", output)

	output = strings.TrimSpace(strings.Replace(output, "\n", " ", -1))

	r.spin.Stop()
	return output
}

func (r Resource) WaitForResourceComplex(timeoutMinutes uint, condition string) error {

	resource := r.ResourceName()

	util.Verbose("Waiting for %s to be %s in the namespace %s for %d minutes\n", resource, condition, r.Namespace, timeoutMinutes)

	r.spin.Prefix = fmt.Sprintf("Waiting for %s to be %s...\t", resource, condition)
	r.spin.Start()
	defer r.spin.Stop()

	var i int
	for start := time.Now(); ; {
		if time.Since(start) > (time.Minute * time.Duration(timeoutMinutes)) {
			return fmt.Errorf("timeout while waiting for %s to be %s", resource, condition)
		} else {
			log, err := osHelper.ProcessCmdResultWithoutLog(*exec.Command("kubectl", "wait",
				"--for", condition,
				resource,
				fmt.Sprintf("--timeout=%ds", timeoutMinutes*60),
				"-n", r.Namespace))
			if err == nil {
				return nil
			} else {
				util.Verbose("Failed waiting for %s to be %s: %s \n%s\n", resource, condition, err.Error(), log)
			}
		}
		time.Sleep(time.Second)
		i++
	}
	return nil
}

func (r Resource) WaitForResource(timeoutMinutes uint, condition string) error {

	resource := r.ResourceName()

	util.Verbose("Waiting for %s to be %s in the namespace %s for %d minutes\n", resource, condition, r.Namespace, timeoutMinutes)

	r.spin.Prefix = fmt.Sprintf("Waiting for %s to be %s...\t", resource, condition)
	r.spin.Start()
	defer r.spin.Stop()

	var i int
	for start := time.Now(); ; {
		if time.Since(start) > (time.Minute * time.Duration(timeoutMinutes)) {
			return fmt.Errorf("timeout while waiting for %s to be %s", resource, condition)
		} else {
			log, err := osHelper.ProcessCmdResultWithoutLog(*exec.Command("kubectl", "wait",
				"--for", fmt.Sprintf("condition=%s", condition),
				resource,
				fmt.Sprintf("--timeout=%ds", timeoutMinutes*60),
				"-n", r.Namespace))
			if err == nil {
				return nil
			} else {
				util.Verbose("Failed waiting for %s to be %s: %s \n%s\n", resource, condition, err.Error(), log)
			}
		}
		time.Sleep(time.Second)
		i++
	}
	return nil
}

func (r Resource) SaveYamlFile(filePath string) error {
	return r.saveYamlFile(r.Name, filePath)
}

func (r Resource) saveYamlFile(anyName interface{}, filePath string) error {

	r.spin.Prefix = fmt.Sprintf("Saving YAML file for %s...\t", r.ResourceName())
	r.spin.Start()
	defer r.spin.Stop()

	r.Command.Args = []string{"get", r.Type, "-n", r.Namespace, "-o", "yaml"}
	if name, status := anyName.(string); status && name != "" {
		r.Command.Args = []string{"get", r.Type, name, "-n", r.Namespace, "-o", "yaml"}
	}

	outfile, err := r.makeFile(filePath)
	if err != nil {
		return fmt.Errorf("error occurred while creating resource %s file %s: %s", r.ResourceName(), filePath, err.Error())
	}
	defer outfile.Close()
	r.Command.StdOut = outfile

	output, ok := r.Command.Run()

	if ok {
		return nil
	} else {
		return fmt.Errorf("error occurred while fetching resource %s: %s", r.ResourceName(), output)
	}
}

func (r Resource) DescribeCommand() string {
	return fmt.Sprintf("kubectl describe %s %s -n %s", r.Type, r.Name, r.Namespace)
}

func (r Resource) SaveDescribeFile(filePath string) error {
	return r.saveDescribeFile(r.Name, filePath)
}

func (r Resource) saveDescribeFile(anyName interface{}, filePath string) error {

	r.spin.Prefix = fmt.Sprintf("Saving describe file for %s...\t", r.ResourceName())
	r.spin.Start()
	defer r.spin.Stop()

	r.Command.Args = []string{"describe", r.Type, "-n", r.Namespace}
	if name, status := anyName.(string); status && name != "" {
		r.Command.Args = []string{"describe", r.Type, name, "-n", r.Namespace}
	}

	outfile, err := r.makeFile(filePath)
	if err != nil {
		return fmt.Errorf("error occurred while creating resource %s file %s: %s", r.ResourceName(), filePath, err.Error())
	}
	defer outfile.Close()
	r.Command.StdOut = outfile

	output, ok := r.Command.Run()

	if ok {
		return nil
	} else {
		return fmt.Errorf("error occurred while fetching resource %s: %s", r.ResourceName(), output)
	}
}

func (r Resource) Filename(suffix string) string {
	return r.filename(r.Name, suffix)
}

func (r Resource) filename(anyName interface{}, suffix string) string {
	if name, status := anyName.(string); status && name != "" {
		return fmt.Sprintf("%s_%s_%s%s", r.Type, name, osHelper.GetDateTime(), suffix)
	} else {
		return fmt.Sprintf("%s_%s%s", r.Type, osHelper.GetDateTime(), suffix)
	}
}

func (r Resource) LogsCommand() string {
	return fmt.Sprintf("kubectl logs %s -n %s -f --all-containers=true", r.ResourceName(), r.Namespace)
}

func (r Resource) SaveLogFile(filePath string, sinceTime int32) error {
	return r.saveLogFile(r.Name, filePath, sinceTime)
}

func (r Resource) saveLogFile(anyName interface{}, filePath string, sinceTime int32) error {

	r.spin.Prefix = fmt.Sprintf("Saving logs file for %s...\t", r.ResourceName())
	r.spin.Start()
	defer r.spin.Stop()

	if sinceTime < 0 {
		sinceTime = 60
	}

	r.Command.Args = []string{"logs", r.Type, "-n", r.Namespace, "--all-containers=true", fmt.Sprintf("--since=%dm", sinceTime)}
	if name, status := anyName.(string); status && name != "" {
		r.Command.Args = []string{"logs", fmt.Sprintf("%s/%s", r.Type, name), "-n", r.Namespace, "--all-containers=true", fmt.Sprintf("--since=%dm", sinceTime)}
	}

	outfile, err := r.makeFile(filePath)
	if err != nil {
		return fmt.Errorf("error occurred while creating resource %s file %s: %s", r.ResourceName(), filePath, err.Error())
	}
	defer outfile.Close()
	r.Command.StdOut = outfile

	output, ok := r.Command.Run()

	if ok {
		return nil
	} else {
		return fmt.Errorf("error occurred while fetching resource %s: %s", r.ResourceName(), output)
	}
}

func (r Resource) makeFile(filePath string) (*os.File, error) {
	path := filepath.Dir(filePath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return nil, err
		}
	}
	return os.Create(filePath)
}

func (r Resource) ResourceName() string {
	resource := r.Type
	if name, status := r.Name.(string); status && name != "" {
		resource = r.Type + "/" + name
	}
	return resource
}
