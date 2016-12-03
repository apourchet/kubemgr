package kubemgr

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/glog"
)

type RawConfig struct {
	Package   string
	Injects   []*Inject
	Resources map[string]*RawResource
}

type SaneConfig struct {
	Package   string
	Injector  Injector
	Resources map[string]*Resource
}

type RawSet []string

func NewRawConfig() *RawConfig {
	c := RawConfig{}
	c.Package = ""
	c.Injects = make([]*Inject, 0)
	c.Resources = make(map[string]*RawResource)
	return &c
}

func NewSaneConfig() *SaneConfig {
	c := SaneConfig{}
	c.Package = ""
	c.Resources = make(map[string]*Resource)
	return &c
}

func (c *RawConfig) FromString(jsonBytes []byte) {
	err := json.Unmarshal(jsonBytes, c)
	if err != nil {
		glog.Errorf("Error parsing mgmt file: %v", err)
		os.Exit(1)
	}
}

func (c *RawConfig) FromFilePath(filePath string) *RawConfig {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		glog.Errorf("Error reading mgmt file %s: %v", filePath, err)
		os.Exit(1)
	}
	c.FromString(content)
	return c
}

func (c *RawConfig) Sanitize() *SaneConfig {
	c.assertConsistent()
	c.closeDeps()

	sane := NewSaneConfig()
	sane.Package = c.Package
	sane.Injector = NewInjector(c.Injects)
	for resName, rawRes := range c.Resources {
		sane.Resources[resName] = FromRawResource(rawRes)
	}

	for resName, res := range sane.Resources {
		raw := c.Resources[resName]
		for _, dep := range raw.Deps {
			res.Deps = append(res.Deps, sane.Resources[dep])
		}
	}

	return sane
}

func (c *RawConfig) Print() {
	content, err := json.MarshalIndent(c, "", "   ")
	if err != nil {
		glog.Infof("Could not marshal rawconfig: %v", err)
		return
	}
	glog.Infof(string(content))
}

func (c *RawConfig) closeDeps() {
	for _, res := range c.Resources {
		allDeps := map[string]interface{}{}
		for i := 0; i < len(res.Deps); i++ {
			dep := res.Deps[i]
			if _, ok := allDeps[dep]; !ok {
				allDeps[dep] = true
				depRes := c.Resources[dep]
				res.Deps = append(res.Deps, depRes.Deps...)
			}
		}
		res.Deps = mapKeys(allDeps)
	}
}

func (c *RawConfig) assertConsistent() {
	for _, res := range c.Resources {
		for _, dep := range res.Deps {
			c.assertHasResource(dep)
		}
	}

	c.assertNoCycles()
}

func (c *RawConfig) assertNoCycles() {
	// TODO implement this
}

func (c *RawConfig) assertHasResource(res string) {
	if _, ok := c.Resources[res]; !ok {
		glog.Errorf("Resource not found: %s", res)
		os.Exit(1)
	}
}

func (c *SaneConfig) Print() {
	content, err := json.MarshalIndent(c, "", "   ")
	if err != nil {
		glog.Infof("Could not marshal saneconfig: %v", err)
		return
	}
	glog.Infof(string(content))
}

func (c *SaneConfig) Apply(target string) {
	for resName, res := range c.Resources {
		if matched, err := resNameMatches(target, resName); matched {
			res.apply(c.Injector)
		} else if err != nil {
			glog.Errorf("Failed to create regex out of target '%s':  %v", target, err)
			os.Exit(1)
		}
	}
}

func (c *SaneConfig) Check(target string) {
	for resName, res := range c.Resources {
		if matched, err := resNameMatches(target, resName); matched {
			res.check()
		} else if err != nil {
			glog.Errorf("Failed to create regex out of target '%s':  %v", target, err)
			os.Exit(1)
		}
	}
}

func (c *SaneConfig) Delete(target string) {
	for resName, res := range c.Resources {
		if matched, err := resNameMatches(target, resName); matched {
			res.delete()
		} else if err != nil {
			glog.Errorf("Failed to create regex out of target '%s':  %v", target, err)
			os.Exit(1)
		}
	}
}

func resNameMatches(target string, resName string) (bool, error) {
	return filepath.Match(target, resName)
}
