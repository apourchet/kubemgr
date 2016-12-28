package kubemgr

import (
	"encoding/json"
	"io/ioutil"

	"github.com/apourchet/kubemgr/lib/kubectl"
	"github.com/golang/glog"
)

type KubeMgr struct {
	filePath string
}

type PackagedContext struct {
	Package string
	Context string
}

func NewKubeMgr(filePath string) *KubeMgr {
	k := KubeMgr{}
	k.filePath = filePath
	return &k
}

func (k *KubeMgr) Do(action string, target string) {
	importManager := NewImportManager()
	resourceManager := NewResourceManager()
	injector := NewInjector()

	// Get and close the import loops
	origImports, err := importManager.GetImports(k.filePath)
	Fatal(err)
	glog.V(3).Infof("Got first level imports: \n   %v", origImports)

	allImports, err := importManager.GetImportClosure(origImports)
	Fatal(err)
	glog.V(3).Infof("Got closed imports: \n   %v", allImports)

	// Get resources from current config
	err = resourceManager.FetchResources(k.filePath)
	Fatal(err)
	glog.V(3).Infof("Got resources: \n%s", resourceManager.String())

	// Get all resources from the imports
	err = resourceManager.GetImportedResources(allImports)
	Fatal(err)
	glog.V(3).Infof("Got imported resources: \n%s", resourceManager.String())

	// Prepare injector
	err = injector.GetInjects(allImports)
	Fatal(err)
	glog.V(3).Infof("Got imported injects: \n%s", injector.String())

	err = injector.GetInjects([]string{k.filePath})
	Fatal(err)
	glog.V(3).Infof("Got injects: \n%s", injector.String())

	// Set the injector on the resourceManager
	err = resourceManager.SetInjector(injector)
	Fatal(err)

	// Check for cyclic dependencies
	err = resourceManager.AssertValid()
	Fatal(err)
	glog.V(1).Infof("Configuration is valid")

	// Set the kubectl context
	glog.V(3).Infof("Reading context from kubeconfig")
	kubectl.Context, err = k.GetContext()
	Fatal(err)

	switch action {
	case ActionInject:
		err = resourceManager.PrepResources(target)
		break
	case ActionApply:
		err = resourceManager.ApplyResources(target)
		break
	case ActionCheck:
		err = resourceManager.CheckResources(target)
		break
	case ActionDelete:
		err = resourceManager.DeleteResources(target)
		break
	case ActionRecreate:
		err = resourceManager.DeleteResources(target)
		if err != nil {
			break
		}
		err = resourceManager.ApplyResources(target)
		break
	}

	Fatal(err)
	glog.V(1).Infof("Done!")
}

func (k *KubeMgr) GetContext() (string, error) {
	configBytes, err := ioutil.ReadFile(k.filePath)
	if err != nil {
		return "", err
	}
	pkg := PackagedContext{}
	err = json.Unmarshal(configBytes, &pkg)
	if err != nil {
		return "", err
	}
	return pkg.Context, nil
}
