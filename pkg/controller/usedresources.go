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
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/beevik/etree"
)

const ProcessDeploymentTopic = "process_deployment"
const ProcessDeploymentParamPrefix = "process_deployment."
const ImportTopic = "import"
const ImportParamPrefix = "import."

// importTypeIdPattern finds the import type id in requests that are no valid json, e.g. because they contain unquoted expressions
var importTypeIdPattern = regexp.MustCompile(`"import_type_id"\s*:\s*("(?:[^"\\]|\\.)*")`)

func parseReleaseUsedResources(xml string) (result model.ReleaseUsedResources, err error) {
	doc := etree.NewDocument()
	err = doc.ReadFromString(xml)
	if err != nil {
		return result, err
	}
	return getReleaseUsedResources(doc), nil
}

func getReleaseUsedResources(doc *etree.Document) model.ReleaseUsedResources {
	processModels := []string{}
	for _, value := range textInputParameters(doc, ProcessDeploymentTopic, ProcessDeploymentParamPrefix+"process_model_id") {
		if isLiteralId(value) {
			processModels = append(processModels, value)
		}
	}
	flows := []string{}
	for _, value := range textInputParameters(doc, AnalyticsTopic, AnalyticsParamPrefix+"flow_id") {
		if isLiteralId(value) {
			flows = append(flows, value)
		}
	}
	importTypes := []string{}
	for _, value := range textInputParameters(doc, ImportTopic, ImportParamPrefix+"request") {
		importTypes = append(importTypes, importTypeIdsOfRequest(value)...)
	}
	return model.ReleaseUsedResources{
		ProcessModels: sortedUnique(processModels),
		Flows:         sortedUnique(flows),
		ImportTypes:   sortedUnique(importTypes),
	}
}

// textInputParameters returns the trimmed text values of the named input parameters of the service tasks with the topic;
// parameters with child elements (scripts, lists, maps) are skipped, since their value is only known at runtime
func textInputParameters(doc *etree.Document, topic string, paramName string) (result []string) {
	for _, task := range doc.FindElements("//bpmn:serviceTask[@camunda:topic='" + topic + "']") {
		for _, param := range task.FindElements(".//camunda:inputParameter") {
			if param.SelectAttrValue("name", "") != paramName || len(param.ChildElements()) > 0 {
				continue
			}
			result = append(result, strings.TrimSpace(param.Text()))
		}
	}
	return result
}

// isLiteralId rejects empty values and values camunda evaluates as expression
func isLiteralId(value string) bool {
	return value != "" && !strings.Contains(value, "${") && !strings.Contains(value, "#{")
}

func importTypeIdsOfRequest(request string) (result []string) {
	parsed := map[string]interface{}{}
	if err := json.Unmarshal([]byte(request), &parsed); err == nil {
		if id, ok := parsed["import_type_id"].(string); ok && isLiteralId(strings.TrimSpace(id)) {
			result = append(result, strings.TrimSpace(id))
		}
		return result
	}
	for _, match := range importTypeIdPattern.FindAllStringSubmatch(request, -1) {
		var id string
		if err := json.Unmarshal([]byte(match[1]), &id); err != nil {
			continue
		}
		if id = strings.TrimSpace(id); isLiteralId(id) {
			result = append(result, id)
		}
	}
	return result
}

func sortedUnique(list []string) []string {
	result := slices.Clone(list)
	if result == nil {
		result = []string{}
	}
	slices.Sort(result)
	return slices.Compact(result)
}

// usedResourcesOf parses the used resources of the release; a bpmn that cannot be parsed yields empty lists marked as unparsable
func (this *Controller) usedResourcesOf(ctx context.Context, release model.SmartServiceReleaseExtended) model.ReleaseUsedResources {
	used, err := parseReleaseUsedResources(release.BpmnXml)
	if err != nil {
		this.config.GetLogger().WarnContext(ctx, "bpmn of release cannot be parsed, it is indexed as using no resources", "releaseId", release.Id, "error", err)
		return model.ReleaseUsedResources{ProcessModels: []string{}, Flows: []string{}, ImportTypes: []string{}, Unparsable: true}
	}
	return used
}

// ensureReleaseUsedResources fills UsedResources of releases stored before the field existed
func (this *Controller) ensureReleaseUsedResources(ctx context.Context, release model.SmartServiceReleaseExtended) model.SmartServiceReleaseExtended {
	if release.UsedResources != nil {
		return release
	}
	used := this.usedResourcesOf(ctx, release)
	release.UsedResources = &used
	return release
}

// BackfillReleaseUsedResources stores the used resources on every release that is not marked as deleted and lacks them,
// release by release; concurrent callers wait for a running backfill instead of reading the same releases in parallel.
// filled counts the releases this call wrote, unparsable those of them whose bpmn could not be parsed.
// On a database error the remaining releases stay without used resources and the next call continues with them.
func (this *Controller) BackfillReleaseUsedResources(ctx context.Context) (filled int, unparsable int, err error) {
	this.usedResourcesBackfillMux.Lock()
	defer this.usedResourcesBackfillMux.Unlock()
	err = this.db.ForEachReleaseWithoutUsedResources(ctx, func(release model.SmartServiceReleaseExtended) error {
		used := this.usedResourcesOf(ctx, release)
		updated, err := this.db.SetReleaseUsedResources(ctx, release.Id, used)
		if err != nil {
			return err
		}
		if updated {
			filled++
			if used.Unparsable {
				unparsable++
			}
		}
		return nil
	})
	return filled, unparsable, err
}

// GetResourceUsage counts the releases and instances of all users that use the resource.
// A release counts if it is not marked as deleted and is the newest of its design or has instances.
// Only the releases the caller may read are named, so no names of foreign releases leak.
func (this *Controller) GetResourceUsage(ctx context.Context, token auth.Token, kind model.ResourceKind, resourceId string) (result model.ResourceUsage, err error, code int) {
	if !kind.Valid() {
		return result, errors.New("unknown resource kind"), http.StatusBadRequest
	}
	resourceId = strings.TrimSpace(resourceId)
	if resourceId == "" {
		return result, errors.New("missing resource id"), http.StatusBadRequest
	}

	// releases without used resources would be missed by the query, so a failed backfill fails the request instead of under-reporting
	_, _, err = this.BackfillReleaseUsedResources(ctx)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	releases, err := this.db.ListReleasesUsingResource(ctx, kind, resourceId)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	releaseIds := []string{}
	successorIds := []string{}
	for _, release := range releases {
		releaseIds = append(releaseIds, release.Id)
		if release.NewReleaseId != "" {
			successorIds = append(successorIds, release.NewReleaseId)
		}
	}
	instanceCounts, err := this.db.CountInstancesOfReleases(ctx, releaseIds)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	// a successor that is marked as deleted or still unfinished may vanish again (deleteRelease() or the cleanup of a failed
	// creation), and the release becomes the newest again; counting it meanwhile errs on the safe side
	liveSuccessorIds, err := this.db.ListFinishedReleaseIds(ctx, successorIds)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	relevant := []model.SmartServiceRelease{}
	relevantIds := []string{}
	for _, release := range releases {
		isNewest := release.NewReleaseId == "" || !slices.Contains(liveSuccessorIds, release.NewReleaseId)
		if isNewest || instanceCounts[release.Id] > 0 {
			relevant = append(relevant, release)
			relevantIds = append(relevantIds, release.Id)
			result.Instances += instanceCounts[release.Id]
		}
	}
	result.Releases = int64(len(relevant))

	result.Readable = []model.ResourceUsageRelease{}
	if len(relevantIds) == 0 {
		return result, nil, http.StatusOK
	}
	access, err, _ := this.permissions.CheckMultiplePermissionsContext(ctx, token.Jwt(), this.config.SmartServiceReleasePermissionsTopic, relevantIds, client.Read)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	for _, release := range relevant {
		if access[release.Id] {
			result.Readable = append(result.Readable, model.ResourceUsageRelease{
				Id:       release.Id,
				DesignId: release.DesignId,
				Name:     release.Name,
			})
		}
	}
	slices.SortFunc(result.Readable, func(a, b model.ResourceUsageRelease) int {
		if c := strings.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		return strings.Compare(a.Id, b.Id)
	})
	return result, nil, http.StatusOK
}
