package kubemgr

import (
	"os"

	"github.com/golang/glog"
)

const (
	ActionApply  = "apply"
	ActionCheck  = "check"
	ActionDelete = "delete"
)

var (
	Actions = map[string]interface{}{
		ActionApply:  true,
		ActionCheck:  true,
		ActionDelete: true,
	}
)

func CheckAction(action string) {
	if _, ok := Actions[action]; !ok {
		glog.Errorf("Failed to perform action '%s': Not Implemented.", action)
		glog.Errorf("Available actions are: %v", mapKeys(Actions))
		os.Exit(1)
	}
}
