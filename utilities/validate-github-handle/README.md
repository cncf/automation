# validate-github-handle

A Google Apps Script library that exposes `VALIDATE_HANDLE(handle)` for Google Sheets. It checks whether a GitHub
handle appears in any of the CNCF community rosters below.

It accepts a bare handle (`handle`), `@handle`, `github.com/handle` or a full `https://github.com/handle` URL, and
ignores case. It returns `VALID | <matches>`, listing only the sources that matched, or `NOT FOUND` if none did.
Unparseable input returns an empty cell.

Example: `VALID | Maintainer: argo | TAG: TAG Infrastructure chair | K8s org`

## What it validates against

| Source | What counts as a match | Output |
|--------|------------------------|--------|
| Project maintainers | Listed in a managed team in `maintainers.yaml` of the project's `.project` repo (`<org>/.project`). Covers every project in the CNCF Landscape, including multi-project orgs. Teams with `managed: false` are excluded. | `Maintainer: <project>` |
| GitHub Enterprise | A member of the CNCF enterprise, in an org belonging to a Graduated or Incubating project. The org set comes from the Landscape. | `GHE: <org>` |
| Kubernetes org | A member or admin in any `config/*/org.yaml` of `kubernetes/org` (for example `kubernetes`, `kubernetes-sigs`). | `K8s org` |
| TOC | Listed as an approver or reviewer in the root `cncf/toc/OWNERS`. | `TOC: <role>` |
| TAG | Listed as a chair, tech lead or TOC liaison in `cncf/toc/tags.yaml`, or as an approver or reviewer in a TAG's `OWNERS` file. | `TAG: <TAG name and role>` |
| CNCF staff | An entry in `cncf/people` `people.json` with category `Staff` and a GitHub handle. | `Staff` |

## Freshness

- Rosters are fetched into an index by `refreshIndex()`, run on a daily trigger, so the data can be up to a day old.
- The GitHub Enterprise check is live (enterprise members GraphQL API) and each result is cached for 6 hours.

## Limitations

- `NOT FOUND` means "not in these sources", not "not a CNCF contributor".
- The root `OWNERS` reviewers list is described in the file as "TOC + shadows", so `TOC` can include shadows.
- Only some `Staff` entries in `people.json` have a GitHub handle, so other staff cannot be matched.
- Project maintainers depend on each project keeping a current `.project/maintainers.yaml`.
- A project's GitHub org is taken from the `repo_url` in the Landscape, which can differ from the org that holds its
  `.project` repo.

## Setup

1. Create a standalone Apps Script project and paste in `validate-github-handle.gs`.
2. In the project's Script Properties set:
   - `GITHUB_TOKEN`: a classic PAT from an enterprise admin with `read:enterprise` and `read:org`, authorized for SSO.
   - `ENTERPRISE_SLUG`: the enterprise slug from `github.com/enterprises/<slug>`.
3. Run `refreshIndex()` and add a daily time-driven trigger for it. The index is stored gzipped in the library's own
   Script Properties.
4. Deploy it as a library (Deploy, New deployment, Library) and share the project with consumers (view access).
   Viewers cannot read Script Properties, so the token stays hidden.
5. In each spreadsheet add the library (identifier `CNCF`) and this wrapper, which is needed because a cell formula
   cannot call a library function directly:

```js
function VALIDATE_HANDLE(handle) {
  return CNCF.VALIDATE_HANDLE(handle);
}
```

Use it as `=VALIDATE_HANDLE(A2)`, or `=VALIDATE_HANDLE(A2:A)` to spill down a column.

If the live GitHub Enterprise lookup fails in a consumer sheet with a permissions error, add the scope
`https://www.googleapis.com/auth/script.external_request` to that sheet's manifest.
