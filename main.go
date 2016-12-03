package main

import (
	"flag"
	"os"

	"github.com/apourchet/kubemgr/lib"
	"github.com/golang/glog"
)

const ()

var (
	action string
	target string

	fname   string
	context string
)

func init() {
	flag.Set("v", "1")
	flag.Set("logtostderr", "true")

	flag.StringVar(&fname, "f", "kubeconfig.json", "Configuration file to use")
	flag.StringVar(&context, "C", "", "Context to execute the command in") // TODO use this
}

func main() {
	checkArgs()
	parseArgs()

	mgr := kubemgr.NewKubeMgr(fname)
	mgr.Do(action, target)
}

func checkArgs() {
	flag.Parse()
	if len(flag.Args()) != 2 {
		glog.Errorf("Not enough arguments")
		os.Exit(1)
	}
}

func parseArgs() {
	action = flag.Args()[0]
	target = flag.Args()[1]
	kubemgr.CheckAction(action)
}
