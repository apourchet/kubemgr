package kubemgr

import (
	"fmt"

	"github.com/golang/glog"
)

type PullMode string

const (
	PullAlways       PullMode = "always"
	PullNever        PullMode = "never"
	PullIfNotPresent PullMode = "ifNotPresent"
)

func pull(path, href string, pullMode PullMode) (string, error) {
	if path == "" && href == "" {
		return "", fmt.Errorf("Must specify either path or href")
	}
	if pullMode == PullNever {
		exists, err := fileExists(path)
		if err != nil {
			glog.Errorf("Failed to find resource '%s': %v", path, err)
			return "", err
		}
		if !exists {
			glog.Errorf("Resource does not exist '%s': %v", path, err)
			return "", fmt.Errorf("Resource file '%s' not found", path)
		}
		return path, nil
	} else if pullMode == PullIfNotPresent || pullMode == "" {
		pathToCheck := path
		if pathToCheck == "" {
			pathToCheck = href
		}
		exists, err := fileExists(pathToCheck)
		if err != nil {
			glog.Errorf("Failed to find resource '%s': %v", path, err)
			return "", err
		}
		if !exists {
			if href == "" {
				glog.Errorf("Failed to pull resource '%s', href empty", path)
				return "", fmt.Errorf("Pull resource '%s' failed: href is empty", path)
			}
			glog.V(2).Infof("Resource '%s' not found; pulling from '%s'", path, href)
			href := fmt.Sprintf("https://%s", href)
			err := wget(href, pathToCheck)
			if err != nil {
				glog.Errorf("Failed to pull resource '%s': %v", path, err)
				return "", err
			}
			return pathToCheck, nil
		}
	} else if pullMode == PullAlways {
		pathToCheck := path
		if pathToCheck == "" {
			pathToCheck = href
		}
		if href == "" {
			glog.Errorf("Failed to pull resource '%s', href empty", path)
			return "", fmt.Errorf("Pull resource '%s' failed: href is empty", path)
		}
		glog.V(2).Infof("Resource '%s' pull set to 'Always'; pulling from '%s'", path, href)
		href := fmt.Sprintf("https://%s", href)
		err := wget(href, pathToCheck)
		if err != nil {
			glog.Errorf("Failed to pull resource '%s': %v", path, err)
			return "", err
		}
		return pathToCheck, nil
	} else {
		return "", fmt.Errorf("Pullmode not recognized: '%s'", pullMode)
	}
	return path, nil
}
