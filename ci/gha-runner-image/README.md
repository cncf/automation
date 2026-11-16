# Kubernetes GHA Image For CNCF Projects

By default, the GitHub runner image does not have certain dependencies (such as `git`). There is an ongoing discussion around this [here](#). For now, we need to build-and-run our own [Image](./Dockerfile)

## Testing a locally-built image before opening a PR

`publish-runner-images.yml` runs a fast in-container smoke test plus a full
ARC/kind end-to-end registration test on every PR. To reproduce either kind
of check yourself, against your own fork/org, without opening a PR:

```sh
docker build -t external-gha-runner:local --build-arg BASE_IMAGE=noble ci/gha-runner-image/

# Full end-to-end: deploy a real ARC runner scale set in a throwaway kind
# cluster and register it against your own repo.
GHA_PAT=ghp_xxx ./.github/scripts/local-kind-runner-test.sh deploy \
  --runner-type container --base-image noble \
  --manifest-file ci/cluster/lke-gha-iad2/manifests/container-runners/cncf-c-ubuntu-2-8-x86.yaml \
  --runner-image external-gha-runner:local \
  --github-repo your-user/automation

./.github/scripts/local-kind-runner-test.sh cleanup
```

The same script also supports `--runner-type vm` for testing
`ci/cloudrunners` image changes against a VM-mode manifest. See the script's
header comment for the full set of flags/environment variables.
