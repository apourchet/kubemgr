package kubemgr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/apourchet/kubemgr/lib/kubectl"
)

type PackagedResources struct {
	Package   string
	Resources map[string]Resource
}

type Resource struct {
	Path string
	Deps []string
}

type ResourceManagerInterface interface {
	FetchResources(filepath string) error
	GetImportedResources(filePaths []string) error
	SetInjector(injector InjectorInterface) error
	ApplyResources(pattern string) error
	CheckResources(pattern string) error
	DeleteResources(pattern string) error
	PrepResources(pattern string) error
	AssertValid() error
	String() string
}

type ResourceManager struct {
	Injector  InjectorInterface
	Resources map[string]Resource
	Prepared  map[string]bool
	Applied   map[string]bool
	Deleted   map[string]bool
}

func NewResourceManager() ResourceManagerInterface {
	r := ResourceManager{}
	r.Injector = nil
	r.Resources = make(map[string]Resource)
	r.Prepared = make(map[string]bool)
	r.Applied = make(map[string]bool)
	r.Deleted = make(map[string]bool)
	return &r
}

func (r *ResourceManager) FetchResources(filepath string) error {
	configBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	pkg := PackagedResources{}
	err = json.Unmarshal(configBytes, &pkg)
	if err != nil {
		return err
	}
	for name, res := range pkg.Resources {
		r.Resources[name] = res
	}
	return nil
}

func (r *ResourceManager) GetImportedResources(filePaths []string) error {
	for _, fpath := range filePaths {
		configBytes, err := ioutil.ReadFile(fpath)
		if err != nil {
			return err
		}
		pkg := PackagedResources{}
		err = json.Unmarshal(configBytes, &pkg)
		if err != nil {
			return err
		}
		prefix := path.Dir(fpath)
		for name, res := range pkg.Resources {
			namespacedName := pkg.Package + "." + name
			prefixedResource := prefixResource(prefix, res)
			r.Resources[namespacedName] = prefixedResource
		}
	}
	return nil
}

func (r *ResourceManager) SetInjector(injector InjectorInterface) error {
	r.Injector = injector
	return nil
}

func (r *ResourceManager) ApplyResources(pattern string) error {
	err := r.PrepResources(pattern)
	if err != nil {
		return err
	}

	resources := r.findAllDependencies(pattern)
	for _, resourceName := range resources {
		if _, found := r.Applied[resourceName]; !found {
			resource := r.Resources[resourceName]
			for _, depName := range resource.Deps {
				err = r.ApplyResources(depName)
				if err != nil {
					return err
				}
			}
			path := r.Injector.GetInjectedFilePath(resource.Path)
			err = kubectl.Apply(path)
			if err != nil {
				return err
			}
			r.Applied[resourceName] = true
		}
	}
	return err
}

func (r *ResourceManager) CheckResources(pattern string) error {
	resources := r.findMatchingResources(pattern)
	for _, resourceName := range resources {
		resource := r.Resources[resourceName]
		path := r.Injector.GetInjectedFilePath(resource.Path)
		err := kubectl.Check(path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ResourceManager) DeleteResources(pattern string) error {
	resources := r.findMatchingResources(pattern)
	for _, resourceName := range resources {
		if _, found := r.Deleted[resourceName]; !found {
			resource := r.Resources[resourceName]
			path := r.Injector.GetInjectedFilePath(resource.Path)
			err := kubectl.Delete(path)
			if err != nil {
				return err
			}
			r.Deleted[resourceName] = true
		}
	}
	return nil
}

func (r *ResourceManager) PrepResources(pattern string) error {
	resources := r.findAllDependencies(pattern)
	for _, resourceName := range resources {
		if _, found := r.Prepared[resourceName]; !found {
			resource := r.Resources[resourceName]
			err := r.Injector.Inject(resource.Path)
			if err != nil {
				return err
			}
			r.Prepared[resourceName] = true
		}
	}
	return nil
}

func (r *ResourceManager) AssertValid() error {
	for resourceName, res := range r.Resources {
		for _, dep := range res.Deps {
			if _, found := r.Resources[dep]; !found {
				return fmt.Errorf("Dependency not found: %s => %s", resourceName, dep)
			}
		}
	}

	// TODO No cycles allowed
	return nil
}

func (r *ResourceManager) String() string {
	content, _ := json.MarshalIndent(r, "", "   ")
	return string(content)
}

func (r *ResourceManager) findMatchingResources(pattern string) []string {
	ret := []string{}
	for resourceName, _ := range r.Resources {
		if match, err := resourceNameMatches(pattern, resourceName); err == nil && match {
			ret = append(ret, resourceName)
		}
	}
	return ret
}

func (r *ResourceManager) findAllDependencies(pattern string) []string {
	allResources := make(map[string]interface{})
	resources := r.findMatchingResources(pattern)
	for _, res := range resources {
		allResources[res] = true
	}

	for i := 0; i < len(resources); i++ {
		resource := r.Resources[resources[i]]
		for _, dep := range resource.Deps {
			if _, found := allResources[resources[i]]; !found {
				resources = append(resources, dep)
				allResources[dep] = true
			}
		}
	}
	for resourceName, _ := range r.Resources {
		if match, err := resourceNameMatches(pattern, resourceName); err == nil && match {
			allResources[resourceName] = true
		}
	}
	return mapKeys(allResources)
}

// ********************
// * HELPER FUNCTIONS *
// ********************
func prefixResource(prefix string, resource Resource) Resource {
	ret := Resource{}
	ret.Path = path.Join(prefix, resource.Path)
	ret.Deps = make([]string, len(resource.Deps))
	for i := range resource.Deps {
		ret.Deps[i] = prefix + resource.Deps[i]
	}
	return ret
}

func resourceNameMatches(target string, resourceName string) (bool, error) {
	return filepath.Match(target, resourceName)
}

// var (
// 	prepared = map[*Resource]bool{}
// 	applied  = map[*Resource]bool{}
// 	deleted  = map[*Resource]bool{}
// )

// func NewResource() *Resource {
// 	r := Resource{}
// 	r.Deps = []*Resource{}
// 	return &r
// }
//
// func FromRawResource(raw *RawResource) *Resource {
// 	res := NewResource()
// 	res.Path = raw.Path
// 	res.Href = raw.Href
// 	res.Pull = raw.Pull
// 	return res
// }
//
// // Pulls the mgmt files according to the pull policy
// func (r *Resource) prep(injector Injector) error {
// 	if _, done := prepared[r]; done {
// 		return nil
// 	}
//
// 	for _, dep := range r.Deps {
// 		err := dep.prep(injector)
// 		if err != nil {
// 			glog.Errorf("Failed preping resource '%s': %v", *r, err)
// 			return err
// 		}
// 	}
//
// 	glog.V(3).Infof("Preping resource: %s", *r)
//
// 	path, err := pull(r.Path, r.Href, r.Pull)
// 	if err != nil {
// 		glog.Errorf("Failed preping resource '%s': %v", *r, err)
// 		return err
// 	}
//
// 	injector.Inject(path)
// 	prepared[r] = true
// 	glog.V(2).Infof("Successfully prepped resource: %s", *r)
// 	return nil
// }
//
// // Applies the resource to the kubernetes cluster
// func (r *Resource) apply(injector Injector) error {
// 	if _, done := applied[r]; done {
// 		return nil
// 	}
//
// 	err := r.prep(injector)
// 	if err != nil {
// 		glog.Errorf("Failed applying resource '%s': %v", *r, err)
// 		return err
// 	}
// 	for _, dep := range r.Deps {
// 		err := dep.apply(injector)
// 		if err != nil {
// 			glog.Errorf("Failed applying resource '%s': %v", *r, err)
// 			return err
// 		}
// 	}
//
// 	glog.V(3).Infof("Applying Resource: %s", *r)
// 	err = kubectl.Apply(r.Path + ".inj")
// 	if err != nil {
// 		glog.Errorf("Failed applying resource '%s': %v", *r, err)
// 		return err
// 	}
//
// 	err = r.check()
// 	if err != nil {
// 		glog.Errorf("Failed applying resource '%s': %v", *r, err)
// 		return err
// 	}
// 	applied[r] = true
// 	glog.V(2).Infof("Successfully applied resource: %s", *r)
// 	return nil
// }
//
// // Checks that this resource is ready to use by
// // the ones that depend on it
// func (r *Resource) check() error {
// 	glog.V(3).Infof("Checking resource: %s", *r)
//
// 	err := kubectl.Check(r.Path + ".inj")
// 	if err != nil {
// 		glog.Errorf("Failed checking resource '%s': %v", err)
// 		return err
// 	}
// 	return err
// }
//
// func (r *Resource) delete() error {
// 	if _, done := deleted[r]; done {
// 		return nil
// 	}
//
// 	glog.V(3).Infof("Deleting resource: %s", *r)
// 	err := kubectl.Delete(r.Path + ".inj")
// 	if err != nil {
// 		glog.Errorf("Failed deleting resource: %s", *r)
// 		return err
// 	}
//
// 	glog.Infof("Successfully deleted resource: %s", *r)
// 	return nil
// }
//
// func (r Resource) String() string {
// 	return r.Path
// }
