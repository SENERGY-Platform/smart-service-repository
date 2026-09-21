# Aspects arrive as a list

A criteria in a smart service design names its aspects in `aspect_ids`. The
former single `aspect_id` still works and is kept as the alias for a list with
one element. This service passes **both spellings on exactly as the design wrote
them** and folds neither — that is the decision this document records, because
the opposite is the obvious thing to do and is wrong here.

## Scope

Applies to `model.Criteria` in `pkg/model/parameter.go`, which is parsed from the
`criteria` and `criteria_list` properties of a camunda `formField`
(`pkg/controller/releaseinfo.go`), stored as part of the release document, and
posted to `POST /v2/query/selectables` of device-selection
(`pkg/selectables/selectables.go`).

**Not this if** the code in hand *evaluates* aspects rather than forwarding
them. A service that has to answer "does this content variable satisfy the
query" folds the deprecated field first and then reads the list only — the
opposite of the rule below, and correct there. Nothing in this repository
evaluates an aspect; it is a parser, a store and a forwarder.

Also not this if the field in hand is `aspect_ids` on a **content variable**.
That one enumerates what a variable carries; a criteria demands that all of them
be carried. Same spelling, opposite direction.

## What a design writes

```json
{"function_id": "urn:infai:ses:measuring-function:...", "aspect_ids": ["urn:infai:ses:aspect:a", "urn:infai:ses:aspect:b"]}
```

Several aspects in **one** criteria are an AND on a single content variable: the
same variable has to carry all of them, each covering its own aspect subtree.
That is a different AND from the one over `criteria_list`, where each entry may
be answered by a different variable of the device type. Two sibling aspects in
one criteria therefore match nothing, because a content variable carries at most
one aspect out of a classified hierarchy.

## Why this service does not fold

The tempting move is to append `aspect_id` into `aspect_ids` at the boundary and
let everything behind it read the list. That is the right move for a service
that consumes a request and discards it. It is wrong here, for one reason:

**A criteria that this service parses is written into the release document and
read back later, by this service and by whatever else reads a release.** Folding
on the way in would put `aspect_ids` into a stored release that a reader
predating the lists cannot see, and clearing the deprecated field would drop the
`aspect_id` that is the only field such a reader looks at. The compatibility the
alias exists to provide would be removed by the code meant to implement it.

The same object is also forwarded on every parameter lookup, where clearing the
deprecated field would be harmless. The stored half decides: a persisted copy is
a later reader even when the same object is also sent somewhere that would not
care.

So the useful question is not "is this a record or a request" but **"is there any
later reader of the field"**. Here there is.

A side effect worth knowing when a test fails: because `aspect_ids` is
`omitempty` in both json and bson, a criteria naming a single aspect the old way
serialises exactly as it did before the lists existed. Releases written before
this change compare byte-identical, which is why the existing fixtures needed no
edit.

## Who resolves the alias

device-selection does, on the receiving side. It decodes the request into its
own filter criteria type, whose `GetAspectIds()` appends the deprecated field,
and passes both spellings on to the device-repository. The version that does
this is `v2.0.2`; an older instance accepts `aspect_ids` and ignores it, which
from the outside is indistinguishable from an aspect that matches nothing. See
[device-selection](https://github.com/SENERGY-Platform/device-selection),
`pkg/controller/devicegroups.go` and `pkg/controller/devicetypeselectable.go`.

## Rejected alternatives

- **Fold on parse, store only the list.** Simplest to read afterwards, and it
  breaks the compatibility guarantee for every reader of a release that does not
  know the list yet.
- **Fold on send, keep the stored form.** Defensible, and it buys nothing: the
  receiver folds anyway, so the only effect would be a second place that has to
  stay in step with the alias rule.
- **A `GetAspectIds()` helper on `model.Criteria`.** Nothing here reads the
  aspects, so it would be a method with no caller, and its existence would
  suggest a read path that does not exist.
