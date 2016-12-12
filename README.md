# kubemgr
A simple tool to templatize and organize kubernetes deployments. Essentially
just a thin wrapper around `kubectl` with some templatizing syntactic sugar,
and resource-dependency management.

### Installation
    go install github.com/apourchet/kubemgr

### Example usage
In this example, we want do deploy an app that depends on a DB deployment as 
well as a syslog service. This means that we want both of those kubernetes
resources available before we create the app deployment.
The syslog service has its own kubemgr configuration file and might depend on 
some other k8s resources, so we need to import it in our kubeconfig.json:
```
"imports": [
    {
        "path": "example_import/kubeconfig.json"
    }
],
```

For resources, we will have the following:
```
"resources": {
    "db-svc": {
        "path": "k8s/db-svc.json"
    },
    "db-dp": {
        "path": "k8s/db-dp.json",
        "deps": ["db-svc"]
    },
    "app-svc": {
        "path": "k8s/app-svc.json"
    },
    "app-dp": {
        "path": "k8s/app-dp.json",
        "deps": ["app-svc", "db-dp", "kubemgr_subtest.syslog-svc"]
    }
}
```

Now if we want to apply our app deployment "app-dp", kubemgr will automatically check
that db-dp has finished deploying, and that app-svc and syslog-svc are both also present.
Thus we can safely apply "app-dp" without worrying about the current state of our cluster.
```
kubemgr apply app-dp
```

Similarly we could also apply all of the resources in our kubeconfig.json file using
regular expressions:
```
kubemgr apply "*"
```

It is also possible to describe dependencies using regexp:
```
"app-dp": {
    "path": "k8s/app-dp.json",
    "deps": ["*-svc", "db-dp"]
}
```

The two actions other than "apply" that are currently supported are "check" and "delete",
which both behave exactly like what you would imagine.

### Injects
Injects are the way you can templatize your k8s files in a more granular way. They contain 
key-value bindings that you can use in your k8s resources. However you also have the benefit
namespacing that you didn't have before. For instance the syslog-svc `NAMESPACE` field is
able to get overridden by the toplevel `injects.json`. You also have the ability to specify
multiple injects with colliding key names that you can then differentiate using that namespacing.

The injected variable's name will have the format: `<packagename>_<injectname>.<varname>`. For 
instance, this inject:
```
"package": "kubemgr_test",
"injects": [
    {
        "name": "mine",
        "path": "injects.json"
    }
]
```
will allow resources to use {{$.kubemgr\_test\_mine.NAMESPACE}}.

### Planned improvements
    Pull resources from the Web
    Check dependency cycles
