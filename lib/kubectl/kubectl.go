package kubectl

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os/exec"
	"time"

	"github.com/golang/glog"
)

type Resource struct {
	Kind   string
	Spec   map[string]interface{}
	Status map[string]interface{}
}

const (
	CheckRetries = 20
	CheckSleep   = 500 * time.Millisecond
)

var (
	Context = ""
)

func init() {
	flag.StringVar(&Context, "context", "", "Kubectl context")
}

func Apply(filePath string) error {
	glog.V(2).Infof("Kubectl applying '%s'", filePath)

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		glog.Errorf("Kubectl failed applying '%s': %v", filePath, err)
		return err
	}
	glog.V(3).Infof("Kubectl applying content: \n%s", string(content))

	args := append([]string{"apply", "-f", filePath}, ContextArgs()...)
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.Output()
	if err != nil {
		glog.Errorf("Kubectl failed applying '%s': %v", filePath, err)
		glog.Errorf("=> %s", string(out))
		return err
	}

	glog.Infof("Kubectl successfully applied '%s' \n=> %s", filePath, string(out))
	return nil
}

func Check(filePath string) error {
	glog.Infof("Kubectl checking '%s'", filePath)

	var err error
	var out []byte
	for i := 0; i < CheckRetries; i++ {
		args := append([]string{"get", "-o", "json", "-f", filePath}, ContextArgs()...)
		out, err = exec.Command("kubectl", args...).Output()
		if err != nil {
			time.Sleep(CheckSleep)
			continue
		}

		res := Resource{}
		err = json.Unmarshal(out, &res)
		if err != nil {
			time.Sleep(CheckSleep)
			continue
		}

		err = res.check()
		if err != nil {
			time.Sleep(CheckSleep)
			continue
		}
	}

	if err != nil {
		glog.Errorf("Kubectl failed checking '%s'", filePath)
		return err
	}

	glog.Infof("Kubectl successfully checked '%s'", filePath)
	return nil
}

func Delete(filePath string) error {
	glog.V(2).Infof("Kubectl deleting '%s'", filePath)

	args := append([]string{"delete", "-f", filePath}, ContextArgs()...)
	out, err := exec.Command("kubectl", args...).Output()
	if err != nil {
		glog.Errorf("Kubectl failed deleting '%s'", filePath)
		return err
	}

	glog.Infof("Kubectl successfully deleted '%s' \n=> %s", filePath, string(out))
	return nil
}

func (r Resource) check() error {
	switch r.Kind {
	case "Service":
		return nil
	case "Deployment":
		want := int(r.Spec["replicas"].(float64))
		haveIf := r.Status["availableReplicas"]
		have := 0
		if haveIf != nil {
			have = int(haveIf.(float64))
		}
		if want == have {
			return nil
		}
		glog.Infof("Want/have %d/%d => waiting and retrying", want, have)
		return fmt.Errorf("Deployment not ready: want %d replicas, has %d.", want, have)
	}
	return nil
}

func ContextArgs() []string {
	if Context == "" {
		return []string{}
	}
	return []string{"--context", Context}
}
