# Release resource changes

A periodic job compares the resources a release uses with the release and records on the release which of them changed or disappeared since it was created. It is only a hint: nothing else on the release changes and no release is created.

## What is checked

The resources come from `used_resources` of the release: process models, flows and import types (see `pkg/controller/usedresources.go`). Each is read with the token of the release creator:

| kind | request | compared |
|---|---|---|
| `process_model` | `GET {process_model_repo_url}/processes/{id}` | `last_updated_unix` > `created_at` of the release |
| `flow` | `GET {flow_repo_url}/flow/{id}` | `dateUpdated` (RFC3339) > `created_at`; missing or zero counts as unknown, not changed |
| `import_type` | `GET {import_repo_url}/import-types/{id}` | sha256 of the response, decoded and re-marshalled with sorted keys, against the baseline stored on the release |

Import types carry no change time. The first run stores the hash as baseline (`import_type_hashes`, stored only, never served); a later difference marks the type as changed. The baseline is not moved, so a changed type stays marked.

Answers:

- 200: compared as above. Only this answer removes an entry from the list.
- 404: the resource is `gone`.
- 401, 403, 5xx, timeout (10 s): logged and counted as error, the previous entry of this resource stays as it is.

## Result

`changed_resources` on `GET /releases`, `GET /releases/{id}` and the extended release endpoints:

```json
{"changed_resources": [{"kind": "flow", "id": "...", "reason": "changed", "at": 1760000000}]}
```

`kind` is `process_model`, `flow` or `import_type`; `reason` is `changed` or `gone`. `at` is the change time reported by the resource. For `gone` and for changed import types it is the time the job first saw it, kept on later runs. The field is absent while nothing is marked. A client cannot set it: `POST /releases` ignores it.

## Configuration

| key | meaning |
|---|---|
| `process_model_repo_url`, `flow_repo_url`, `import_repo_url` | base url of the repository; empty skips that kind, its entries on the releases stay as they are |
| `resource_check_cycle` | interval, `1h` in `config.json`; empty switches the job off |

The job runs once at start in the background and then every cycle. Runs never overlap. Each run logs one line with the releases checked and marked and the number of errors. A release is written only when its list or baseline changed, by a targeted update (`SetReleaseResourceCheck`).

## Known limits

- Publishing a process model bumps its time, so a release is marked although the content may be the same.
- Import types are compared from the first run on; changes before that are not seen.
- Resources the creator can no longer read are not reported; the previous entry stays.
- `SetRelease` replaces the whole document, so a release read before a run and written afterwards drops the result of that run. The next run records it again, but a lost import type baseline is stored anew from the then current content. Accepted, as for `used_resources`.
