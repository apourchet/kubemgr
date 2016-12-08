package kubemgr

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io/ioutil"
	"os"
	"path"

	"github.com/golang/glog"
)

type Inject struct {
	Name string
	Path string
}

type PackagedInjects struct {
	Package string
	Injects []Inject
}

type InjectorInterface interface {
	GetInjects(configPaths []string) error
	Inject(filepath string) error
	GetInjectedFilePath(filePath string) string
	String() string
}

type Injector struct {
	Injects []Inject
	Data    map[string]interface{}
}

func NewInjector() InjectorInterface {
	i := Injector{}
	i.Data = make(map[string]interface{})
	return &i
}

func (injector *Injector) GetInjects(configPaths []string) error {
	injects, err := fetchInjects(configPaths)
	if err != nil {
		return err
	}
	for _, i := range injects {
		data, err := dataFromFile(i.Path)
		if err != nil {
			return err
		}

		innerData := make(map[string]interface{})
		for k, v := range data {
			injector.Data[k] = v // Global
			innerData[k] = v     // Namespaced
		}
		injector.Data[i.Name] = innerData
	}
	return nil
}

// Injects a file and outputs the new injected file's path
func (i *Injector) Inject(filepath string) error {
	in, err := ioutil.ReadFile(filepath)
	if err != nil {
		glog.Errorf("Failed to inject file '%s': %v", err)
		return err
	}

	out, err := i.doInject(in)
	if err != nil {
		glog.Errorf("Failed to inject file '%s': %v", err)
		return err
	}

	outfname := i.GetInjectedFilePath(filepath)
	err = ioutil.WriteFile(outfname, out, 0644)
	if err != nil {
		glog.Errorf("Failed to write injected file '%s': %v", err)
		return err
	}

	glog.Infof("Successfully injected '%s'=> '%s'", filepath, outfname)
	return nil
}

func (i *Injector) GetInjectedFilePath(filePath string) string {
	return filePath + ".inj"
}

func (i *Injector) doInject(content []byte) ([]byte, error) {
	tname := fmt.Sprintf("%s", sha1.Sum(content))
	tmpl, err := template.New(tname).Funcs(getFuncMap()).Parse(string(content))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, i.Data)
	if err != nil {
		return nil, err
	}
	str := html.UnescapeString(buf.String())
	glog.V(3).Infof("Injected contents: \n%s", str)
	return []byte(str), nil
}

func (i *Injector) String() string {
	content, _ := json.MarshalIndent(i, "", "   ")
	return string(content)
}

func fetchInjects(configPaths []string) ([]Inject, error) {
	injects := []Inject{}
	for _, fpath := range configPaths {
		configBytes, err := ioutil.ReadFile(fpath)
		if err != nil {
			return nil, err
		}
		pkg := PackagedInjects{}
		err = json.Unmarshal(configBytes, &pkg)
		if err != nil {
			return nil, err
		}
		prefix := path.Dir(fpath)
		for i := range pkg.Injects {
			pkg.Injects[i].Name = pkg.Package + "-" + pkg.Injects[i].Name
			pkg.Injects[i].Path = path.Join(prefix, pkg.Injects[i].Path)
		}
		injects = append(injects, pkg.Injects...)
	}
	return injects, nil
}

// *************************************
// Helper functions for the templating *
// *************************************
func dataFromFile(filepath string) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	configBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return data, err
	}
	err = json.Unmarshal(configBytes, &data)
	if err != nil {
		return data, err
	}
	err = objectsToString(data)
	return data, nil
}

func readFile(fname string) (string, error) {
	b, err := ioutil.ReadFile(fname)
	if err != nil {
		return "", err
	}
	return string(b), err
}

func base64Encode(s string) string {
	return string(base64.StdEncoding.EncodeToString([]byte(s)))
}

func quote(s string) string {
	return fmt.Sprintf("%q", s)
}

func loopOverInts(n float64) []int {
	arr := make([]int, int(n))
	for i := 0; i < int(n); i++ {
		arr[i] = i
	}
	return arr
}

func getFuncMap() template.FuncMap {
	return template.FuncMap{
		"include": readFile,
		"base64":  base64Encode,
		"loop":    loopOverInts,
		"quote":   quote,
	}
}

func objectsToString(m map[string]interface{}) error {
	for k, v := range m {
		cast, ok := v.(map[string]interface{})
		if ok {
			strBytes, err := json.Marshal(cast)
			if err != nil {
				return err
			}
			m[k] = string(strBytes)
			continue
		}
		cast1, ok := v.([]interface{})
		if ok {
			strBytes, err := json.Marshal(cast1)
			if err != nil {
				return err
			}
			m[k] = string(strBytes)
			continue
		}
	}
	return nil
}

func envOverride(m map[string]interface{}) {
	for k, _ := range m {
		if os.Getenv(k) != "" {
			m[k] = os.Getenv(k)
		}
	}
}
