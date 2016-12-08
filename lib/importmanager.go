package kubemgr

import (
	"encoding/json"
	"io/ioutil"
	"path"
)

type PackagedImports struct {
	Package string
	Imports []Import
}

type Import struct {
	Path string
}

type ImportManagerInterface interface {
	GetImports(filepath string) ([]string, error)
	GetImportClosure([]string) ([]string, error)
}

type ImportManager struct{}

func NewImportManager() ImportManagerInterface {
	i := ImportManager{}
	return &i
}

func (i *ImportManager) GetImports(filepath string) ([]string, error) {
	configBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	pkg := PackagedImports{}
	err = json.Unmarshal(configBytes, &pkg)
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(pkg.Imports))
	for i := range pkg.Imports {
		paths[i] = pkg.Imports[i].Path
	}
	return paths, nil
}

func (mgr *ImportManager) GetImportClosure(imports []string) ([]string, error) {
	for i := 0; i < len(imports); i++ {
		imp := imports[i]
		subImps, err := mgr.GetImports(imp)
		if err != nil {
			return nil, err
		}
		prefix := path.Dir(imp)
		newImps := pathsFromImports(prefix, subImps)
		imports = append(imports, newImps...)
	}
	return imports, nil
}

func pathsFromImports(prefix string, imports []string) []string {
	ret := make([]string, len(imports))
	for i := range imports {
		ret[i] = path.Join(prefix, imports[i])
	}
	return ret
}
