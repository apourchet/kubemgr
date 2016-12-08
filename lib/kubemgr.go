package kubemgr

import "github.com/golang/glog"

type KubeMgr struct {
	filePath string
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

	switch action {
	case ActionApply:
		err = resourceManager.ApplyResources(target)
	case ActionCheck:
		err = resourceManager.CheckResources(target)
	case ActionDelete:
		err = resourceManager.DeleteResources(target)
	}

	Fatal(err)
	glog.V(1).Infof("Done!")
}
