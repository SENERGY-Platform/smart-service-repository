/*
 * Copyright (c) 2022 InfAI (CC SES)
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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	permmodel "github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

func (this *Controller) retryMarkedReleases() {
	toDelete, unfinised, err := this.db.GetMarkedReleases()
	if err != nil {
		this.config.GetLogger().Error("error in retryMarkedReleases", "error", err)
		return
	}
	for _, release := range toDelete {
		err = this.deleteRelease(release.Id)
		if err != nil {
			this.config.GetLogger().Error("error in retryMarkedReleases()::deleteRelease()", "error", err, "releaseId", release.Id)
			return
		}
	}
	for _, release := range unfinised {
		err = this.deleteRelease(release.Id)
		if err != nil {
			this.config.GetLogger().Error("error in retryMarkedReleases()::deleteRelease()", "error", err, "releaseId", release.Id)
			return
		}
	}
}

func (this *Controller) CreateRelease(token auth.Token, element model.SmartServiceRelease) (result model.SmartServiceRelease, err error, code int) {
	if element.DesignId == "" {
		return result, errors.New("missing design id"), http.StatusBadRequest
	}
	design, err, code := this.GetDesign(token, element.DesignId)
	if err != nil {
		if code == http.StatusNotFound {
			return result, fmt.Errorf("user does not own a smart-service-design with the id %v", element.DesignId), http.StatusBadRequest
		}
		return result, err, http.StatusInternalServerError
	}
	if element.Name == "" {
		element.Name = design.Name
	}
	if element.Description == "" {
		element.Description = design.Description
	}
	element.CreatedAt = time.Now().Unix()
	element.Creator = token.GetUserId()

	if element.Id == "" {
		element.Id = this.GetNewId()
	}

	err = ValidateDesign(this.config, design.BpmnXml)
	if err != nil {
		return result, fmt.Errorf("invalid design xml for release: %w", err), http.StatusBadRequest
	}

	parsedInfo, err := this.parseDesignXmlForReleaseInfo(token, design.BpmnXml, element)
	if err != nil {
		return result, fmt.Errorf("unable to parse design xml for release: %w", err), http.StatusBadRequest
	}
	err = this.validateParsedReleaseInfos(parsedInfo)
	if err != nil {
		return result, err, http.StatusBadRequest
	}

	err = this.saveReleaseCreate(model.SmartServiceReleaseExtended{
		SmartServiceRelease: element,
		BpmnXml:             design.BpmnXml,
		SvgXml:              design.SvgXml,
		ParsedInfo:          parsedInfo,
	})
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	return element, nil, http.StatusOK
}

func (this *Controller) saveReleaseCreate(release model.SmartServiceReleaseExtended) (err error) {
	if release.Creator == "" {
		return errors.New("missing creator")
	}
	err, _ = this.db.SetRelease(release, true)
	if err != nil {
		return err
	}

	err = this.deployRelease(release)
	if err != nil {
		temperr := this.deleteRelease(release.Id)
		if temperr != nil {
			this.config.GetLogger().Warn("error while rolling back deployRelease(); will be retired", "releaseId", release.Id, "error", temperr)
		}
		return err
	}
	err = this.db.MarkReleaseAsFinished(release.Id)
	if err != nil {
		return err
	}
	return nil
}

func (this *Controller) getInitialReleasePermissions(release model.SmartServiceReleaseExtended, oldReleases []model.SmartServiceReleaseExtended) (permissionAlreadyExists bool, initialPermissions client.ResourcePermissions, err error) {
	defaultInitialPermissions := client.ResourcePermissions{
		UserPermissions: map[string]permmodel.PermissionsMap{
			release.Creator: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			},
		},
	}

	token, err := this.adminAccess.EnsureAccess(this.config)
	if err != nil {
		this.config.GetLogger().Warn("error in getInitialReleasePermissions", "error", err, "stack", string(debug.Stack()))
		return permissionAlreadyExists, initialPermissions, err
	}
	perm, err, code := this.permissions.GetResource(token, this.config.SmartServiceReleasePermissionsTopic, release.Id)
	if err == nil {
		return true, perm.ResourcePermissions, nil
	}
	if code != http.StatusNotFound && code != http.StatusForbidden {
		return permissionAlreadyExists, initialPermissions, err
	}

	sort.Slice(oldReleases, func(i, j int) bool {
		return oldReleases[i].CreatedAt > oldReleases[i].CreatedAt
	})
	if len(oldReleases) > 0 {
		perm, err, code = this.permissions.GetResource(token, this.config.SmartServiceReleasePermissionsTopic, oldReleases[len(oldReleases)-1].Id)
		if err == nil {
			perm.UserPermissions[release.Creator] = client.PermissionsMap{
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			}
			return false, perm.ResourcePermissions, nil
		}
		if code != http.StatusNotFound && code != http.StatusForbidden {
			this.config.GetLogger().Warn("unable to get permission of old releases, fall back to default initial permissions", "error", err)
			return false, defaultInitialPermissions, nil
		}
	}
	return false, defaultInitialPermissions, nil
}

func (this *Controller) getOldReleases(release model.SmartServiceReleaseExtended) (result []model.SmartServiceReleaseExtended, err error) {
	oldReleases, err := this.db.GetReleasesByDesignId(release.DesignId)
	if err != nil {
		return result, err
	}
	for _, oldRelease := range oldReleases {
		if oldRelease.Id != release.Id {
			result = append(result, oldRelease)
		}
	}
	return result, nil
}

func (this *Controller) deployRelease(release model.SmartServiceReleaseExtended) (err error) {
	oldReleases := []model.SmartServiceReleaseExtended{}
	if release.NewReleaseId == "" {
		oldReleases, err = this.getOldReleases(release)
	}
	permAlreadyExist, initialPermissions, err := this.getInitialReleasePermissions(release, oldReleases)
	if err != nil {
		return err
	}
	if !permAlreadyExist {
		_, err, _ = this.permissions.SetPermission(client.InternalAdminToken, this.config.SmartServiceReleasePermissionsTopic, release.Id, initialPermissions)
		if err != nil {
			return err
		}
	}

	err, _ = this.camunda.DeployRelease(release.Creator, release)
	if err != nil {
		return err
	}

	for _, old := range oldReleases {
		if old.CreatedAt < release.CreatedAt && old.Id != release.Id { //"if" to prevent race from  HandleReleaseDelete() to recreate deleted release
			instances, err, _ := this.db.ListInstancesOfRelease("", old.Id)
			if err != nil {
				return err
			}
			if len(instances) == 0 && this.config.DeleteUnusedOldVersionReleases {
				err = this.deleteRelease(old.Id)
				if err != nil {
					return err
				}
			} else {
				old.NewReleaseId = release.Id
				err, _ = this.db.SetRelease(old, false)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (this *Controller) GetRelease(token auth.Token, id string) (result model.SmartServiceRelease, err error, code int) {
	access, err, _ := this.permissions.CheckPermission(token.Jwt(), this.config.SmartServiceReleasePermissionsTopic, id, client.Read)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !access {
		return result, errors.New("access denied"), http.StatusForbidden
	}
	var extended model.SmartServiceReleaseExtended
	extended, err, code = this.db.GetRelease(id, false)
	return extended.SmartServiceRelease, err, code
}

func (this *Controller) ListReleases(token auth.Token, query model.ReleaseQueryOptions) (result []model.SmartServiceRelease, total int64, err error, code int) {
	temp, total, err, code := this.ListExtendedReleases(token, query)
	if err != nil {
		return nil, 0, err, code
	}
	for _, release := range temp {
		result = append(result, release.SmartServiceRelease)
	}
	return result, total, nil, http.StatusOK
}

func (this *Controller) GetExtendedRelease(token auth.Token, id string) (result model.SmartServiceReleaseExtended, err error, code int) {
	access, err, _ := this.permissions.CheckPermission(token.Jwt(), this.config.SmartServiceReleasePermissionsTopic, id, client.Read)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !access {
		return result, errors.New("access denied"), http.StatusForbidden
	}
	result, err, code = this.db.GetRelease(id, false)
	if err != nil {
		return result, err, code
	}
	result, err = this.ensureValidReleaseModuleInfo(result)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return result, nil, http.StatusOK
}

func (this *Controller) ListExtendedReleases(token auth.Token, query model.ReleaseQueryOptions) (result []model.SmartServiceReleaseExtended, total int64, err error, code int) {
	checkedRigths, err := permmodel.PermissionListFromString(query.Rights)
	if err != nil {
		return result, 0, err, http.StatusBadRequest
	}
	listOptions := client.ListOptions{}
	if len(query.Ids) > 0 {
		listOptions.Ids = query.Ids
	}
	ids, err, _ := this.permissions.ListAccessibleResourceIds(token.Jwt(), this.config.SmartServiceReleasePermissionsTopic, listOptions, checkedRigths...)
	if err != nil {
		return result, 0, err, http.StatusInternalServerError
	}
	if len(query.Ids) > 0 {
		ids_t := []string{}
		for _, id := range query.Ids {
			if slices.Contains(ids, id) {
				ids_t = append(ids_t, id)
			}
		}
		ids = ids_t
	}
	temp, total, err := this.db.ListReleases(model.ListReleasesOptions{
		InIds:  ids,
		Latest: query.Latest,
		Limit:  query.Limit,
		Offset: query.Offset,
		Sort:   query.GetSort(),
		Search: query.Search,
	})
	if err != nil {
		return result, 0, err, http.StatusInternalServerError
	}
	filteredIds := []string{}
	for _, release := range temp {
		filteredIds = append(filteredIds, release.Id)
	}
	permWrapper, err, _ := this.permissions.ListComputedPermissions(token.Jwt(), this.config.SmartServiceReleasePermissionsTopic, filteredIds)

	permissionsIndex := map[string]map[string]bool{}
	for _, perm := range permWrapper {
		permissionsIndex[perm.Id] = computedPermissionsToMap(perm)
	}
	for _, release := range temp {
		release.PermissionsInfo = model.PermissionsInfo{
			Shared:      token.GetUserId() != release.Creator,
			Permissions: permissionsIndex[release.Id],
		}
		release, err = this.ensureValidReleaseModuleInfo(release)
		if err != nil {
			return result, total, err, http.StatusInternalServerError
		}
		result = append(result, release)
	}
	return result, total, nil, http.StatusOK
}

func computedPermissionsToMap(perm permmodel.ComputedPermissions) map[string]bool {
	return map[string]bool{
		"r": perm.Read,
		"w": perm.Write,
		"x": perm.Execute,
		"a": perm.Administrate,
	}
}

func (this *Controller) DeleteRelease(token auth.Token, releaseId string, deletePreviousReleases bool) (error, int) {
	ids := []string{}

	if deletePreviousReleases {
		previous, err := this.db.GetPreviousReleases(releaseId)
		if err != nil {
			return err, http.StatusInternalServerError
		}

		for _, p := range previous {
			ids = append(ids, p.Id)
		}
	}
	ids = append(ids, releaseId) // ensure delete this release last for best performance: no replacement of NewReleaseId required

	accessMap, err, _ := this.permissions.CheckMultiplePermissions(token.Jwt(), this.config.SmartServiceReleasePermissionsTopic, ids, client.Administrate)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	for _, access := range accessMap {
		if !access {
			return errors.New("access denied"), http.StatusForbidden
		}
	}

	for _, id := range ids {
		instances, err, code := this.db.ListInstancesOfRelease("", id)
		if err != nil {
			return err, code
		}
		if len(instances) > 0 {
			list := []map[string]string{}
			for _, instance := range instances {
				list = append(list, map[string]string{
					"id":   instance.Id,
					"user": instance.UserId,
				})
			}
			marshaledList, _ := json.Marshal(list)
			err = fmt.Errorf("a release may only deleted if it is not referenced by any smart-service instance: %s", marshaledList)
			return err, http.StatusBadRequest
		}
	}

	for _, id := range ids {
		err = this.deleteRelease(id)
		if err != nil {
			return err, http.StatusInternalServerError
		}
	}

	return nil, http.StatusOK
}

func (this *Controller) deleteRelease(id string) error {
	err, _ := this.db.MarlReleaseAsDeleted(id) //to enable retry if permissions.RemoveResource() fails
	if err != nil {
		return err
	}

	//remove release from camunda
	err = this.camunda.RemoveRelease(id)
	if err != nil {
		return err
	}

	//update NewReleaseId on other releases if this release is the newest one
	currentRelease, err, code := this.db.GetRelease(id, true)
	if err != nil && code != http.StatusNotFound {
		return err
	}
	if err == nil && currentRelease.NewReleaseId == "" {
		oldReleases, err := this.db.GetPreviousReleases(currentRelease.Id)
		if err != nil {
			return err
		}
		sort.Slice(oldReleases, func(i, j int) bool {
			return oldReleases[i].CreatedAt > oldReleases[j].CreatedAt
		})
		youngestRelease := model.SmartServiceReleaseExtended{}
		for _, value := range oldReleases {
			if value.Id == currentRelease.Id {
				continue
			}
			if youngestRelease.Id == "" {
				youngestRelease = value
				break
			}
		}
		if youngestRelease.Id != "" {
			youngestRelease.NewReleaseId = ""
			err = this.saveReleaseCreate(youngestRelease)
			if err != nil {
				return err
			}
		}
		//other releases will be updated on update handling of youngestRelease because NewReleaseId == ""
		//there is a race between the deletion of this release from the database and the update of releases that are not youngestRelease in HandleReleaseSave()
		//but the retroactive create/uptdate of the release that is meant to be deleted is prevented by "if old.CreatedAt < release.CreatedAt {" in HandleReleaseSave()
	}

	//delete release from db
	err, _ = this.permissions.RemoveResource(client.InternalAdminToken, this.config.SmartServiceReleasePermissionsTopic, id)
	if err != nil {
		this.config.GetLogger().Warn("permissions.RemoveResource() failed but will be retried", "topic", this.config.SmartServiceReleasePermissionsTopic, "releaseId", id, "error", err)
		return nil
	}
	err, _ = this.db.DeleteRelease(id)
	if err != nil {
		this.config.GetLogger().Warn("db.DeleteRelease() failed but will be retried", "releaseId", id, "error", err)
		return nil
	}
	return nil
}

func (this *Controller) GetReleaseParameter(token auth.Token, id string) (result []model.SmartServiceExtendedParameter, err error, code int) {
	access, err, _ := this.permissions.CheckPermission(token.Jwt(), this.config.SmartServiceReleasePermissionsTopic, id, client.Execute)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !access {
		return result, errors.New("access denied"), http.StatusForbidden
	}
	return this.GetReleaseParameterWithoutAuthCheck(token, id)
}

func (this *Controller) parameterDescriptionsToSmartServiceExtendedParameter(token auth.Token, paramList []model.ParameterDescription) (result []model.SmartServiceExtendedParameter, err error, code int) {
	for _, paramDesc := range paramList {
		if paramDesc.AutoSelectAll {
			continue //will be filled on instantiation of the release
		}
		param := model.SmartServiceExtendedParameter{
			SmartServiceParameter: model.SmartServiceParameter{
				Id:    paramDesc.Id,
				Value: nil,
				Label: paramDesc.Label,
			},
			Description:      paramDesc.Description,
			DefaultValue:     paramDesc.DefaultValue,
			Type:             getSchemaOrgType(paramDesc.Type),
			Multiple:         paramDesc.Multiple,
			Order:            paramDesc.Order,
			CharacteristicId: paramDesc.CharacteristicId,
			Characteristic:   paramDesc.Characteristic,
			Optional:         paramDesc.Optional,
		}
		param.Options, err, code = this.getParamOptions(token, paramDesc)
		if err != nil {
			return result, err, code
		}
		param.HasNoValidOption = !(param.Optional || paramDesc.IotDescription == nil || len(param.Options) > 0)

		//set default value to nil if it cont be found in options
		if len(param.Options) > 0 && param.DefaultValue != nil {
			found := false
			for _, option := range param.Options {
				if option.Value == param.DefaultValue {
					found = true
					break
				}
			}
			if !found {
				param.DefaultValue = nil
			}
		}

		//sort options
		sort.Slice(param.Options, func(i, j int) bool {
			return strings.ToLower(param.Options[i].Label) < strings.ToLower(param.Options[j].Label)
		})

		result = append(result, param)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Order < result[j].Order
	})
	return result, nil, http.StatusOK
}

func (this *Controller) GetReleaseParameterWithoutAuthCheck(token auth.Token, id string) (result []model.SmartServiceExtendedParameter, err error, code int) {
	release, err, code := this.db.GetRelease(id, false)
	if err != nil {
		return result, err, code
	}
	return this.parameterDescriptionsToSmartServiceExtendedParameter(token, release.ParsedInfo.ParameterDescriptions)
}

func getSchemaOrgType(t string) model.Type {
	switch t {
	case "boolean":
		return model.Boolean
	case "string":
		return model.String
	case "long":
		return model.Integer
	case "number":
		return model.Float
	default:
		return model.Type(t)
	}
}
