package kubectl

import (
	"io/ioutil"
	"os/exec"

	"github.com/golang/glog"
)

func Apply(filePath string) error {
	glog.V(3).Infof("Kubectl applying '%s'", filePath)

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		glog.Errorf("Kubectl failed applying '%s': %v", filePath, err)
		return err
	}
	glog.V(3).Infof("Kubectl applying content: \n%s", string(content))

	out, err := exec.Command("kubectl", "apply", "-f", filePath).Output()
	if err != nil {
		glog.Errorf("Kubectl failed applying '%s': %v", filePath, err)
		return err
	}

	glog.Infof("Kubectl successfully applied '%s' \n=> %s", filePath, string(out))
	return nil
}

func Check(filePath string) error {
	glog.V(3).Infof("Kubectl checking '%s'", filePath)

	out, err := exec.Command("kubectl", "get", "--no-headers=true", "-f", filePath).Output()
	if err != nil {
		glog.Errorf("Kubectl failed checking '%s'", filePath)
		return err
	}

	glog.Infof("Kubectl successfully checked '%s' \n=> %s", filePath, string(out))
	return nil
}

func Delete(filePath string) error {
	glog.V(3).Infof("Kubectl deleting '%s'", filePath)

	out, err := exec.Command("kubectl", "delete", "-f", filePath).Output()
	if err != nil {
		glog.Errorf("Kubectl failed deleting '%s'", filePath)
		return err
	}

	glog.Infof("Kubectl successfully deleted '%s' \n=> %s", filePath, string(out))
	return nil
}
