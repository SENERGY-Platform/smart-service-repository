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

const resourceCheckTimeout = 10 * time.Second   //one request
const resourceCheckReleaseTimeout = time.Minute //all requests of one release

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
	// all releases are read first: a slow repository must not hold a cursor open or make a later run skip the same releases again
	releases, err := this.db.ListReleasesWithUsedResources(ctx)
	if err != nil {
		return result, err
	}
	releaseTimeout := resourceCheckReleaseTimeout
	if this.releaseCheckTimeout > 0 {
		releaseTimeout = this.releaseCheckTimeout
	}
	for _, release := range releases {
		if err = ctx.Err(); err != nil {
			return result, err
		}
		releaseCtx, cancel := context.WithTimeout(ctx, releaseTimeout)
		marked, errCount := this.checkReleaseResources(releaseCtx, release)
		cancel()
		result.Checked++
		result.Errors += errCount
		if marked {
			result.Marked++
		}
	}
	return result, nil
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

// observeResource reads the resource with the token of the release creator; 404 means gone.
// The repositories check the permission before the existence, and a deleted resource loses its permission entry, so a deleted
// resource answers 401 or 403 like one the creator may not read. Such an answer is repeated once with the token of the service,
// which tells gone (404) from unreadable (200 with the resource).
// Any other answer or a failed request leaves the state unknown and is returned as error.
func (this *Controller) observeResource(ctx context.Context, lookup resourceLookup, userJwt string) (result resourceObservation, err error) {
	body, status, err := this.readResource(ctx, lookup, userJwt)
	if err == nil && (status == http.StatusUnauthorized || status == http.StatusForbidden) {
		var adminJwt string
		adminJwt, err = this.adminAuthorization()
		if err != nil {
			return result, fmt.Errorf("unable to get service token: %w", err)
		}
		body, status, err = this.readResource(ctx, lookup, adminJwt)
	}
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
		result.hash, err = importTypeHash(body)
		if err != nil {
			return result, err
		}
	}
	result.state = resourceFound
	return result, nil
}

func (this *Controller) readResource(ctx context.Context, lookup resourceLookup, jwt string) (body []byte, status int, err error) {
	var path string
	switch lookup.kind {
	case model.ChangedResourceKindProcessModel:
		path = "/processes/"
	case model.ChangedResourceKindFlow:
		path = "/flow/"
	case model.ChangedResourceKindImportType:
		path = "/import-types/"
	default:
		return nil, 0, fmt.Errorf("unknown resource kind %v", lookup.kind)
	}
	return getResourceBody(ctx, lookup.baseUrl+path+url.PathEscape(lookup.id), jwt)
}

// the fields of an import type that decide what an import does; the hash covers only these, so new fields of the api
// and metadata like owner or cost do not mark releases
type importTypeContent struct {
	Name           string                `json:"name"`
	Description    string                `json:"description"`
	Image          string                `json:"image"`
	DefaultRestart bool                  `json:"default_restart"`
	Configs        []importTypeConfig    `json:"configs,omitempty"`
	Output         importContentVariable `json:"output"`
}

type importTypeConfig struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Type         string      `json:"type"`
	DefaultValue interface{} `json:"default_value"`
}

type importContentVariable struct {
	Name                string                  `json:"name"`
	Type                string                  `json:"type"`
	CharacteristicId    string                  `json:"characteristic_id"`
	SubContentVariables []importContentVariable `json:"sub_content_variables,omitempty"`
	UseAsTag            bool                    `json:"use_as_tag"`
	FunctionId          string                  `json:"function_id,omitempty"`
	AspectId            string                  `json:"aspect_id,omitempty"` // only read: folded into AspectIds
	AspectIds           []string                `json:"aspect_ids,omitempty"`
}

func (this *importContentVariable) normalize() {
	if len(this.AspectIds) == 0 && this.AspectId != "" {
		this.AspectIds = []string{this.AspectId}
	}
	this.AspectId = ""
	for i := range this.SubContentVariables {
		this.SubContentVariables[i].normalize()
	}
}

func importTypeHash(body []byte) (string, error) {
	var content importTypeContent
	if err := json.Unmarshal(body, &content); err != nil {
		return "", err
	}
	content.Output.normalize()
	normalized, err := json.Marshal(content)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(normalized)
	return hex.EncodeToString(sum[:]), nil
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
