package kubemgr

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/apourchet/kubemgr/lib/kubectl"
	"github.com/golang/glog"
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

var (
	SkipDeps bool
)

func init() {
	flag.BoolVar(&SkipDeps, "skip-deps", false, "Skip the dependencies")
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
				if SkipDeps {
					continue
				}
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
			err = kubectl.Check(path)
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
				glog.Warningf("Error: %v", err)
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
			if len(r.findMatchingResources(dep)) <= 0 {
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

	if SkipDeps {
		return mapKeys(allResources)
	}

	for i := 0; i < len(resources); i++ {
		resource := r.Resources[resources[i]]
		for _, dep := range resource.Deps {
			if _, found := allResources[dep]; !found {
				resources = append(resources, dep)
				allResources[dep] = true
			}
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
