package kubemgr

import (
	"io"
	"net/http"
	"os"

	"github.com/golang/glog"
)

func fileExists(filePath string) (bool, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false, nil
	} else if err == nil {
		return true, nil
	} else {
		return false, err
	}
}

func wget(href string, filePath string) error {
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(href)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func mapKeys(m map[string]interface{}) []string {
	res := make([]string, 0)
	for k, _ := range m {
		res = append(res, k)
	}
	return res
}

func mergeMaps(m1, m2 map[*Resource]bool) map[*Resource]bool {
	for k, _ := range m2 {
		m1[k] = true
	}
	return m1
}

func Fatal(err error) {
	if err != nil {
		glog.Fatalf("Error: %v", err)
	}
}
