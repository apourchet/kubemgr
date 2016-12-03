package kubemgr

import (
	"fmt"

	"github.com/apourchet/kubemgr/lib/kubectl"
	"github.com/golang/glog"
)

type RawResource struct {
	Path string
	Href string
	Pull PullMode
	Deps []string
}

type Resource struct {
	Path string
	Href string
	Pull PullMode
	Deps []*Resource
}

type PullMode string

const (
	PullAlways       PullMode = "always"
	PullNever        PullMode = "never"
	PullIfNotPresent PullMode = "ifNotPresent"
)

var (
	prepared = map[*Resource]bool{}
	applied  = map[*Resource]bool{}
	deleted  = map[*Resource]bool{}
)

func NewResource() *Resource {
	r := Resource{}
	r.Deps = []*Resource{}
	return &r
}

func FromRawResource(raw *RawResource) *Resource {
	res := NewResource()
	res.Path = raw.Path
	res.Href = raw.Href
	res.Pull = raw.Pull
	return res
}

func (r *Resource) pull() (string, error) {
	if r.Path == "" && r.Href == "" {
		return "", fmt.Errorf("Failed to pull resource '%s': Must specify either path or href", *r)
	}
	if r.Pull == PullNever {
		exists, err := fileExists(r.Path)
		if err != nil {
			glog.Errorf("Failed to find resource '%s': %v", r.Path, err)
			return "", err
		}
		if !exists {
			glog.Errorf("Resource does not exist '%s': %v", r.Path, err)
			return "", fmt.Errorf("Resource file '%s' not found", r.Path)
		}
		return r.Path, nil
	} else if r.Pull == PullIfNotPresent || r.Pull == "" {
		pathToCheck := r.Path
		if pathToCheck == "" {
			pathToCheck = r.Href
		}
		exists, err := fileExists(pathToCheck)
		if err != nil {
			glog.Errorf("Failed to find resource '%s': %v", r.Path, err)
			return "", err
		}
		if !exists {
			if r.Href == "" {
				glog.Errorf("Failed to pull resource '%s', href empty", r.Path)
				return "", fmt.Errorf("Pull resource '%s' failed: href is empty", r.Path)
			}
			glog.V(2).Infof("Resource '%s' not found; pulling from '%s'", r.Path, r.Href)
			href := fmt.Sprintf("https://%s", r.Href)
			err := wget(href, pathToCheck)
			if err != nil {
				glog.Errorf("Failed to pull resource '%s': %v", r.Path, err)
				return "", err
			}
		}
	} else if r.Pull == PullAlways {
		pathToCheck := r.Path
		if pathToCheck == "" {
			pathToCheck = r.Href
		}
		if r.Href == "" {
			glog.Errorf("Failed to pull resource '%s', href empty", r.Path)
			return "", fmt.Errorf("Pull resource '%s' failed: href is empty", r.Path)
		}
		glog.V(2).Infof("Resource '%s' pull set to 'Always'; pulling from '%s'", r.Path, r.Href)
		href := fmt.Sprintf("https://%s", r.Href)
		err := wget(href, pathToCheck)
		if err != nil {
			glog.Errorf("Failed to pull resource '%s': %v", r.Path, err)
			return "", err
		}
	}

	return r.Path, nil
}

// Pulls the mgmt files according to the pull policy
func (r *Resource) prep(injector Injector) error {
	if _, done := prepared[r]; done {
		return nil
	}

	for _, dep := range r.Deps {
		err := dep.prep(injector)
		if err != nil {
			glog.Errorf("Failed preping resource '%s': %v", *r, err)
			return err
		}
	}

	glog.V(3).Infof("Preping resource: %s", *r)

	path, err := r.pull()
	if err != nil {
		glog.Errorf("Failed preping resource '%s': %v", *r, err)
		return err
	}

	injector.Inject(path)
	prepared[r] = true
	glog.V(2).Infof("Successfully prepped resource: %s", *r)
	return nil
}

// Applies the resource to the kubernetes cluster
func (r *Resource) apply(injector Injector) error {
	if _, done := applied[r]; done {
		return nil
	}

	err := r.prep(injector)
	if err != nil {
		glog.Errorf("Failed applying resource '%s': %v", *r, err)
		return err
	}
	for _, dep := range r.Deps {
		err := dep.apply(injector)
		if err != nil {
			glog.Errorf("Failed applying resource '%s': %v", *r, err)
			return err
		}
	}

	glog.V(3).Infof("Applying Resource: %s", *r)
	err = kubectl.Apply(r.Path + ".inj")
	if err != nil {
		glog.Errorf("Failed applying resource '%s': %v", *r, err)
		return err
	}

	err = r.check()
	if err != nil {
		glog.Errorf("Failed applying resource '%s': %v", *r, err)
		return err
	}
	applied[r] = true
	glog.V(2).Infof("Successfully applied resource: %s", *r)
	return nil
}

// Checks that this resource is ready to use by
// the ones that depend on it
func (r *Resource) check() error {
	glog.V(3).Infof("Checking resource: %s", *r)

	err := kubectl.Check(r.Path + ".inj")
	if err != nil {
		glog.Errorf("Failed checking resource '%s': %v", err)
		return err
	}
	return err
}

func (r *Resource) delete() error {
	if _, done := deleted[r]; done {
		return nil
	}

	glog.V(3).Infof("Deleting resource: %s", *r)
	err := kubectl.Delete(r.Path + ".inj")
	if err != nil {
		glog.Errorf("Failed deleting resource: %s", *r)
		return err
	}

	glog.Infof("Successfully deleted resource: %s", *r)
	return nil
}

func (r Resource) String() string {
	return r.Path
}
