kube2consul
===========

Install
-------

Get the binary directly from GitHub releases or download the code and compile it with `make`. It requires Go 1.5 or later.


Usage
-----

| Command line option | Environment option   | Default value             |
| ------------------- | -------------------- | ------------------------- |
| `-consul-api`       | `K2C_CONSUL_API`     | `"127.0.0.1:8500"`        |
| `-consul-token`     | `K2C_TOKEN_API`      | `""`                      |
| `-kubernetes-api`   | `K2C_KUBERNETES_API` | `"http://127.0.0.1:8080"` |
| `-resync-period`    | `K2C_RESYNC_PERIOD`  | `30`                      |
