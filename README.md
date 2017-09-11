kube2consul
===========

Install
-------

Get the binary directly from GitHub releases or download the code and compile it with `make`. It requires Go 1.8 or later.


Usage
-----
Kube2consul runs in a kubernetes cluster by default. It is able to work out of cluster if an absolute path to the kubeconfig file is provided.

| Command line option | Environment option   | Default value             |
| ------------------- | -------------------- | ------------------------- |
| `-consul-api`       | `K2C_CONSUL_API`     | `"127.0.0.1:8500"`        |
| `-consul-token`     | `K2C_CONSUL_TOKEN`   | `""`                      |
| `-kubernetes-api`   | `K2C_KUBERNETES_API` | `""`                      |
| `-resync-period`    | `K2C_RESYNC_PERIOD`  | `30`                      |
| `-kubeconfig`       | `K2C_KUBECONFIG`     | `""`                      |
