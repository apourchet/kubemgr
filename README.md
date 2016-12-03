# kubemgr
A simple tool to templatize and organize kubernetes deployments. Essentially
just a thin wrapper around `kubectl` with some templatizing syntactic sugar,
and resource-dependency management.

### Installation
    go install github.com/apourchet/kubemgr

### Dependencies
You now have a way to describe the dependencies that each resource 
depend on. `kubemgr` will prioritize those dependencies when 
applying them.

### Usage
    kubemgr apply deployment1
    kubemgr delete deployment1
