# Kubernetes GHA Image For CNCF Projects

By default, the GitHub runner image does not have certain dependencies (such as `git`). There is an ongoing discussion around this [here](#). For now, we need to build-and-run our own [Image](./Dockerfile)
