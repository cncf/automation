## Secrets

Regardless of the provider, this secret will be expected in the namespace for GitHub's ARC to work properly:

```
apiVersion: v1
data:
  github_token: nahhhhhh
kind: Secret
metadata:
  name: github-arc-secret
  namespace: arc-systems
type: Opaque
```