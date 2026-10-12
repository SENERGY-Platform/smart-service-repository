/*
 * Copyright 2026 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/SENERGY-Platform/gin-middleware/otelx"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

const resourceCheckTimeout = 10 * time.Second

type resourceState int

const (
	resourceUnknown resourceState = iota // not readable right now: the previous entry stays
	resourceFound
	resourceGone
)

// resourceObservation is what the check learned about one resource; changeTime is set for process models and flows, hash for import types
type resourceObservation struct {
	state      resourceState
	changeTime int64
	hash       string
}

// ResourceCheckResult counts the releases a run looked at, the ones that list changed resources afterwards, and the failed reads and writes
type ResourceCheckResult struct {
	Checked int
	Marked  int
	Errors  int
}

func (this *Controller) nowUnix() int64 {
	if this.clock != nil {
		return this.clock().Unix()
	}
	return time.Now().Unix()
}

// CheckReleaseResources compares the resources every release uses with the release and records on the release which of them
// changed or are gone; concurrent callers wait for a running check.
// It is only a hint: nothing else on the release changes.
func (this *Controller) CheckReleaseResources(ctx context.Context) (result ResourceCheckResult, err error) {
	this.resourceCheckMux.Lock()
	defer this.resourceCheckMux.Unlock()
	err = this.db.ForEachReleaseWithUsedResources(ctx, func(release model.SmartServiceReleaseExtended) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		result.Checked++
		marked, errCount := this.checkReleaseResources(ctx, release)
		result.Errors += errCount
		if marked {
			result.Marked++
		}
		return nil
	})
	return result, err
}

// checkReleaseResources reports whether the release is marked afterwards and how many reads or writes failed
func (this *Controller) checkReleaseResources(ctx context.Context, release model.SmartServiceReleaseExtended) (marked bool, errCount int) {
	marked = len(release.ChangedResources) > 0
	if release.UsedResources == nil {
		return marked, 0
	}
	lookups := []resourceLookup{}
	lookups = this.appendLookups(lookups, model.ChangedResourceKindProcessModel, this.config.ProcessModelRepoUrl, release.UsedResources.ProcessModels)
	lookups = this.appendLookups(lookups, model.ChangedResourceKindFlow, this.config.FlowRepoUrl, release.UsedResources.Flows)
	lookups = this.appendLookups(lookups, model.ChangedResourceKindImportType, this.config.ImportRepoUrl, release.UsedResources.ImportTypes)
	if !slices.ContainsFunc(lookups, func(lookup resourceLookup) bool { return lookup.baseUrl != "" }) {
		return marked, 0
	}

	token, err := this.userTokenProvider(release.Creator)
	if err != nil {
		this.config.GetLogger().WarnContext(ctx, "resource check: unable to get token of release creator", "releaseId", release.Id, "error", err)
		return marked, 1
	}

	previous := map[string]model.ChangedResource{}
	for _, entry := range release.ChangedResources {
		previous[resourceKey(entry.Kind, entry.Id)] = entry
	}
	hashes := maps.Clone(release.ImportTypeHashes)
	if hashes == nil {
		hashes = map[string]string{}
	}
	changed := []model.ChangedResource{}
	for _, lookup := range lookups {
		old, hasOld := previous[resourceKey(lookup.kind, lookup.id)]
		observation := resourceObservation{}
		if lookup.baseUrl != "" {
			var err error
			observation, err = this.observeResource(ctx, lookup, token.Jwt())
			if err != nil {
				this.config.GetLogger().WarnContext(ctx, "resource check: unable to read resource, keep its previous state", "releaseId", release.Id, "kind", lookup.kind, "resourceId", lookup.id, "error", err)
				errCount++
			}
		}
		switch observation.state {
		case resourceUnknown:
			if hasOld {
				changed = append(changed, old)
			}
		case resourceGone:
			at := this.nowUnix()
			if hasOld && old.Reason == model.ChangedResourceReasonGone {
				at = old.At
			}
			changed = append(changed, model.ChangedResource{Kind: lookup.kind, Id: lookup.id, Reason: model.ChangedResourceReasonGone, At: at})
		case resourceFound:
			if lookup.kind == model.ChangedResourceKindImportType {
				baseline, hasBaseline := hashes[lookup.id]
				if !hasBaseline {
					hashes[lookup.id] = observation.hash
				} else if baseline != observation.hash {
					at := this.nowUnix()
					if hasOld && old.Reason == model.ChangedResourceReasonChanged {
						at = old.At
					}
					changed = append(changed, model.ChangedResource{Kind: lookup.kind, Id: lookup.id, Reason: model.ChangedResourceReasonChanged, At: at})
				}
			} else if observation.changeTime > 0 && observation.changeTime > release.CreatedAt {
				changed = append(changed, model.ChangedResource{Kind: lookup.kind, Id: lookup.id, Reason: model.ChangedResourceReasonChanged, At: observation.changeTime})
			}
		}
	}

	if slices.Equal(changed, release.ChangedResources) && maps.Equal(hashes, release.ImportTypeHashes) {
		return marked, errCount
	}
	err = this.db.SetReleaseResourceCheck(ctx, release.Id, changed, hashes)
	if err != nil {
		this.config.GetLogger().ErrorContext(ctx, "resource check: unable to store result", "releaseId", release.Id, "error", err)
		return marked, errCount + 1
	}
	return len(changed) > 0, errCount
}

// resourceLookup is one used resource; an empty baseUrl marks a kind the check skips, whose previous entries stay as they are
type resourceLookup struct {
	kind    string
	id      string
	baseUrl string
}

func (this *Controller) appendLookups(list []resourceLookup, kind string, baseUrl string, ids []string) []resourceLookup {
	for _, id := range ids {
		list = append(list, resourceLookup{kind: kind, id: id, baseUrl: strings.TrimRight(baseUrl, "/")})
	}
	return list
}

func resourceKey(kind string, id string) string {
	return kind + "/" + id
}

// observeResource reads the resource with the token of the release creator;
// 404 means gone, any other non-200 answer or a failed request leaves the state unknown and is returned as error
func (this *Controller) observeResource(ctx context.Context, lookup resourceLookup, jwt string) (result resourceObservation, err error) {
	var path string
	switch lookup.kind {
	case model.ChangedResourceKindProcessModel:
		path = "/processes/"
	case model.ChangedResourceKindFlow:
		path = "/flow/"
	case model.ChangedResourceKindImportType:
		path = "/import-types/"
	default:
		return result, fmt.Errorf("unknown resource kind %v", lookup.kind)
	}
	body, status, err := getResourceBody(ctx, lookup.baseUrl+path+url.PathEscape(lookup.id), jwt)
	if err != nil {
		return result, err
	}
	if status == http.StatusNotFound {
		result.state = resourceGone
		return result, nil
	}
	if status != http.StatusOK {
		return result, fmt.Errorf("unexpected response: %v", status)
	}
	switch lookup.kind {
	case model.ChangedResourceKindProcessModel:
		var process struct {
			LastUpdatedUnix int64 `json:"last_updated_unix"`
		}
		if err = json.Unmarshal(body, &process); err != nil {
			return result, err
		}
		result.changeTime = process.LastUpdatedUnix
	case model.ChangedResourceKindFlow:
		var flow struct {
			DateUpdated string `json:"dateUpdated"`
		}
		if err = json.Unmarshal(body, &flow); err != nil {
			return result, err
		}
		if flow.DateUpdated != "" {
			updated, err := time.Parse(time.RFC3339, flow.DateUpdated)
			if err != nil {
				return result, err
			}
			if !updated.IsZero() {
				result.changeTime = updated.Unix()
			}
		}
	case model.ChangedResourceKindImportType:
		var importType map[string]interface{}
		if err = json.Unmarshal(body, &importType); err != nil {
			return result, err
		}
		normalized, err := json.Marshal(importType) // map keys are sorted, so the hash does not depend on the key order of the response
		if err != nil {
			return result, err
		}
		sum := sha256.Sum256(normalized)
		result.hash = hex.EncodeToString(sum[:])
	}
	result.state = resourceFound
	return result, nil
}

func getResourceBody(ctx context.Context, target string, jwt string) (body []byte, status int, err error) {
	ctx, cancel := context.WithTimeout(ctx, resourceCheckTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, 0, err
	}
	err = otelx.InjectContextToRequest(ctx, req)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", jwt)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}
