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
	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	permmodel "github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/beevik/etree"
	"log"
	"net/http"
	"runtime/debug"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (this *Controller) retryMarkedReleases() {
	toDelete, unfinised, err := this.db.GetMarkedReleases()
	if err != nil {
		log.Println("ERROR: retryMarkedReleases()", err)
		return
	}
	for _, release := range toDelete {
		err = this.deleteRelease(release.Id)
		if err != nil {
			log.Println("ERROR: retryMarkedReleases()::deleteRelease()", release.Id, err)
			return
		}
	}
	for _, release := range unfinised {
		err = this.deleteRelease(release.Id)
		if err != nil {
			log.Println("ERROR: retryMarkedReleases()::deleteRelease()", release.Id, err)
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

	err = ValidateDesign(design.BpmnXml)
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
	oldReleases := []model.SmartServiceReleaseExtended{}
	if release.NewReleaseId == "" {
		oldReleases, err = this.db.GetReleasesByDesignId(release.DesignId)
		if err != nil {
			return err
		}
	}
	sort.Slice(oldReleases, func(i, j int) bool {
		return oldReleases[i].CreatedAt > oldReleases[i].CreatedAt
	})
	if len(oldReleases) > 0 {
		err = this.copyRightsOfRelease(release.Creator, oldReleases[len(oldReleases)-1], release)
		if err != nil {
			return err
		}
	}

	err, _ = this.db.SetRelease(release, false)
	if err != nil {
		return err
	}

	err = this.deployRelease(release, oldReleases)
	if err != nil {
		temperr := this.deleteRelease(release.Id)
		if temperr != nil {
			log.Println("WARNING: error while rolling back deployRelease(); will be retired", release.Id, temperr)
		}
		return err
	}
	err = this.db.MarkReleaseAsFinished(release.Id)
	if err != nil {
		return err
	}
	return nil
}

func (this *Controller) deployRelease(release model.SmartServiceReleaseExtended, oldReleases []model.SmartServiceReleaseExtended) (err error) {
	if oldReleases == nil {
		oldReleases = []model.SmartServiceReleaseExtended{}
		if release.NewReleaseId == "" {
			oldReleases, err = this.db.GetReleasesByDesignId(release.DesignId)
			if err != nil {
				return err
			}
		}
	}

	_, err, _ = this.permissions.SetPermission(client.InternalAdminToken, this.config.SmartServiceReleasePermissionsTopic, release.Id, client.ResourcePermissions{
		UserPermissions: map[string]permmodel.PermissionsMap{
			release.Creator: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			}},
	})
	if err != nil {
		return err
	}

	err, _ = this.camunda.DeployRelease(release.Creator, release)
	if err != nil {
		return err
	}

	for _, old := range oldReleases {
		if old.CreatedAt < release.CreatedAt && old.Id != release.Id { //"if" to prevent race from  HandleReleaseDelete() to recreate deleted release
			old.NewReleaseId = release.Id
			err = this.saveReleaseCreate(old)
			if err != nil {
				return err
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

func (this *Controller) ListReleases(token auth.Token, query model.ReleaseQueryOptions) (result []model.SmartServiceRelease, err error, code int) {
	temp, err, code := this.ListExtendedReleases(token, query)
	if err != nil {
		return nil, err, code
	}
	for _, release := range temp {
		result = append(result, release.SmartServiceRelease)
	}
	return result, nil, http.StatusOK
}

func (this *Controller) GetExtendedRelease(token auth.Token, id string) (result model.SmartServiceReleaseExtended, err error, code int) {
	access, err, _ := this.permissions.CheckPermission(token.Jwt(), this.config.SmartServiceReleasePermissionsTopic, id, client.Read)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !access {
		return result, errors.New("access denied"), http.StatusForbidden
	}
	return this.db.GetRelease(id, false)
}

func (this *Controller) ListExtendedReleases(token auth.Token, query model.ReleaseQueryOptions) (result []model.SmartServiceReleaseExtended, err error, code int) {
	checkedRigths, err := permmodel.PermissionListFromString(query.Rights)
	if err != nil {
		return result, err, http.StatusBadRequest
	}
	ids, err, _ := this.permissions.ListAccessibleResourceIds(token.Jwt(), this.config.SmartServiceReleasePermissionsTopic, client.ListOptions{}, checkedRigths...)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	temp, err := this.db.ListReleases(model.ListReleasesOptions{
		InIds:  ids,
		Latest: query.Latest,
		Limit:  query.Limit,
		Offset: query.Offset,
		Sort:   query.GetSort(),
		Search: query.Search,
	})
	if err != nil {
		return result, err, http.StatusInternalServerError
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
		result = append(result, release)
	}
	return result, nil, http.StatusOK
}

func computedPermissionsToMap(perm permmodel.ComputedPermissions) map[string]bool {
	return map[string]bool{
		"r": perm.Read,
		"w": perm.Write,
		"x": perm.Execute,
		"a": perm.Administrate,
	}
}

func (this *Controller) DeleteRelease(token auth.Token, releaseId string) (error, int) {
	access, err, _ := this.permissions.CheckPermission(token.Jwt(), this.config.SmartServiceReleasePermissionsTopic, releaseId, client.Administrate)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	if !access {
		return errors.New("access denied"), http.StatusForbidden
	}

	instances, err, code := this.db.ListInstancesOfRelease("", releaseId)
	if err != nil {
		return err, code
	}
	if len(instances) > 0 {
		return errors.New("a release may only deleted if it is not referenced by any smart-service instance"), http.StatusBadRequest
	}

	err = this.deleteRelease(releaseId)
	if err != nil {
		return err, http.StatusInternalServerError
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
		oldReleases, err := this.db.GetReleasesByDesignId(currentRelease.DesignId)
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
		log.Println("WARNING: permissions.RemoveResource() failed but will be retried", this.config.SmartServiceReleasePermissionsTopic, id, err)
		return nil
	}
	err, _ = this.db.DeleteRelease(id)
	if err != nil {
		log.Println("WARNING: db.DeleteRelease() failed but will be retried", id, err)
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

func (this *Controller) copyRightsOfRelease(owner string, oldRelease model.SmartServiceReleaseExtended, newRelease model.SmartServiceReleaseExtended) error {
	token, err := this.adminAccess.EnsureAccess(this.config)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return err
	}
	rights, err, _ := this.permissions.GetResource(token, this.config.SmartServiceReleasePermissionsTopic, oldRelease.Id)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return err
	}
	rights.UserPermissions[owner] = client.PermissionsMap{
		Read:         true,
		Write:        true,
		Execute:      true,
		Administrate: true,
	}
	_, err, _ = this.permissions.SetPermission(token, this.config.SmartServiceReleasePermissionsTopic, newRelease.Id, rights.ResourcePermissions)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return err
	}
	return nil
}

//------------ Parsing ----------------

func (this *Controller) parseDesignXmlForReleaseInfo(token auth.Token, xml string, element model.SmartServiceRelease) (result model.SmartServiceReleaseInfo, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			log.Printf("%s: %s", r, debug.Stack())
			err = errors.New(fmt.Sprint("Recovered Error: ", r))
		}
	}()
	doc := etree.NewDocument()
	err = doc.ReadFromString(xml)
	if err != nil {
		return result, err
	}
	startEvents := doc.FindElements("//bpmn:startEvent")
	for _, startEvent := range startEvents {
		parameter := []model.ParameterDescription{}
		for _, formField := range startEvent.FindElements(".//camunda:formField") {
			id := formField.SelectAttrValue("id", "")
			if id == "" {
				return result, errors.New("missing id in camunda:formField")
			}
			label := formField.SelectAttrValue("label", id)
			if label == "" {
				label = id
			}
			fieldType := formField.SelectAttrValue("type", "")
			if fieldType == "" {
				return result, errors.New("missing type in camunda:fieldType")
			}
			var defaultValue interface{}
			defaultValueField := formField.SelectAttr("defaultValue")
			if defaultValueField != nil {
				switch fieldType {
				case "string":
					defaultValue = defaultValueField.Value
				case "long":
					defaultValue, err = strconv.ParseFloat(defaultValueField.Value, 64)
					if err != nil {
						return result, fmt.Errorf("expect number in camunda:formField %v defaultValue: %w", id, err)
					}
				case "boolean":
					defaultValue, err = strconv.ParseBool(defaultValueField.Value)
					if err != nil {
						return result, fmt.Errorf("expect boolean in camunda:formField %v defaultValue: %w", id, err)
					}
				}

			}
			properties := map[string]string{}

			for _, property := range formField.FindElements("./camunda:properties/camunda:property") {
				propertyId := property.SelectAttrValue("id", "")
				if propertyId == "" {
					return result, fmt.Errorf("missing property id in formField %v", id)
				}
				properties[propertyId] = property.SelectAttrValue("value", "")
			}

			param := model.ParameterDescription{
				Id:           id,
				Label:        label,
				Description:  properties["description"],
				Type:         fieldType,
				DefaultValue: defaultValue,
				Optional:     strings.ToLower(strings.TrimSpace(properties["optional"])) == "true",
			}
			if order, ok := properties["order"]; ok {
				param.Order, err = strconv.Atoi(order)
				if err != nil {
					return result, fmt.Errorf("invalid order property for formField %v: %w", id, err)
				}
			}
			if chId, ok := properties["characteristic_id"]; ok {
				param.CharacteristicId = &chId
				param.Characteristic, err = this.GetCharacteristic(chId)
				if err != nil {
					return result, fmt.Errorf("unable to find characteristics_id for formField %v: %w", id, err)
				}
			}
			if options, ok := properties["options"]; ok {
				if _, containsCharacteristic := properties["characteristic_id"]; containsCharacteristic {
					return result, fmt.Errorf("invalid characteristics_id/options property for formField %v: %v", id, "options and characteristics_id are mutual exclusive")
				}
				err = json.Unmarshal([]byte(options), &param.Options)
				if err != nil {
					return result, fmt.Errorf("invalid options property for formField %v: %w", id, err)
				}
			}
			if multiple, ok := properties["multiple"]; ok {
				param.Multiple, err = strconv.ParseBool(multiple)
				if err != nil {
					return result, fmt.Errorf("invalid multiple property for formField %v: %w", id, err)
				}
			}
			if autoSelectAll, ok := properties["auto_select_all"]; ok {
				param.AutoSelectAll, err = strconv.ParseBool(autoSelectAll)
				if err != nil {
					return result, fmt.Errorf("invalid auto_select_all property for formField %v: %w", id, err)
				}
				if param.AutoSelectAll && !param.Multiple {
					return result, fmt.Errorf("auto_select_all property may only be used in combination with multiple for formField %v: %w", id, err)
				}
			}
			if iot, ok := properties["iot"]; ok {
				if _, containsOptions := properties["options"]; containsOptions {
					return result, fmt.Errorf("invalid options/iot property for formField %v: %v", id, "iot and options are mutual exclusive")
				}
				if _, containsCharacteristic := properties["characteristic_id"]; containsCharacteristic {
					return result, fmt.Errorf("invalid characteristics_id/iot property for formField %v: %v", id, "iot and characteristics_id are mutual exclusive")
				}
				typeFilter := []string{}
				iot = strings.ReplaceAll(iot, " ", "")
				if iot != "" {
					typeFilter = strings.Split(iot, ",")
				}
				criteria := []model.Criteria{}
				if criteriaStr, hasCriteria := properties["criteria"]; hasCriteria {
					temp := model.Criteria{}
					err = json.Unmarshal([]byte(criteriaStr), &temp)
					if err != nil {
						return result, fmt.Errorf("invalid criteria property for formField %v: %w", id, err)
					}
					criteria = []model.Criteria{temp}
				}
				if criteriaStr, hasCriteria := properties["criteria_list"]; hasCriteria {
					err = json.Unmarshal([]byte(criteriaStr), &criteria)
					if err != nil {
						return result, fmt.Errorf("invalid criteria property for formField %v: %w", id, err)
					}
				}

				entityOnly := false
				if entityOnlyStr, hasEntityOnly := properties["entity_only"]; hasEntityOnly {
					entityOnly, _ = strconv.ParseBool(entityOnlyStr)
				}

				sameEntity := properties["same_entity"]

				param.IotDescription = &model.IotDescription{
					TypeFilter:                   typeFilter,
					Criteria:                     criteria,
					EntityOnly:                   entityOnly,
					NeedsSameEntityIdInParameter: sameEntity,
				}
			}
			parameter = append(parameter, param)
		}
		if isMaintenanceProcedure(startEvent) {
			maintenanceProcedure, err := createMaintenanceProcedure(doc, startEvent, element, parameter)
			if err != nil {
				return result, err
			}
			result.MaintenanceProcedures = append(result.MaintenanceProcedures, maintenanceProcedure)
		} else {
			result.ParameterDescriptions = parameter
		}
	}

	return result, nil
}

func createMaintenanceProcedure(doc *etree.Document, event *etree.Element, element model.SmartServiceRelease, parameter []model.ParameterDescription) (result model.MaintenanceProcedure, err error) {
	result.ParameterDescriptions = parameter
	result.BpmnId = event.SelectAttrValue("id", "")
	msgEvent := event.FindElement(".//bpmn:messageEventDefinition")
	if msgEvent == nil {
		return result, fmt.Errorf("missing bpmn:messageEventDefinition for %v", result.BpmnId)
	}
	result.MessageRef = msgEvent.SelectAttrValue("messageRef", "")
	if result.MessageRef == "" {
		return result, fmt.Errorf("missing messageRef for %v", result.BpmnId)
	}
	msgRefElement := doc.FindElement("//bpmn:message[@id='" + result.MessageRef + "']")
	if msgRefElement == nil {
		return result, fmt.Errorf("unknown messageRef %v for %v", result.MessageRef, result.BpmnId)
	}
	result.PublicEventId = msgRefElement.SelectAttrValue("name", "")
	result.InternalEventId = element.Id + "_" + result.PublicEventId

	return result, nil
}

func isMaintenanceProcedure(element *etree.Element) bool {
	msgEvent := element.FindElement(".//bpmn:messageEventDefinition")
	if msgEvent == nil {
		return false //is not msg event
	}
	return true
}

func (this *Controller) validateParsedReleaseInfos(info model.SmartServiceReleaseInfo) error {
	knownInitParams := map[string]bool{}
	for _, param := range info.ParameterDescriptions {
		if ok := knownInitParams[param.Id]; ok {
			return fmt.Errorf("reuse of %v as param-id", param.Id)
		}
		knownInitParams[param.Id] = true
		if param.AutoSelectAll && !param.Multiple {
			return fmt.Errorf("%v: parameter property \"auto_select_all\" may only be used in combination with  \"multiple\"", param.Id)
		}
		if param.IotDescription != nil {
			if param.IotDescription.NeedsSameEntityIdInParameter != "" {
				if !slices.ContainsFunc(info.ParameterDescriptions, func(p model.ParameterDescription) bool {
					return p.Id == param.IotDescription.NeedsSameEntityIdInParameter
				}) {
					return fmt.Errorf("%v: parameter property \"entity_only\" references unknown parameter \"%v\"", param.Id, param.IotDescription.NeedsSameEntityIdInParameter)
				}
			}
		}
	}
	knownEventIds := map[string]bool{}
	for _, maintenanceProcedure := range info.MaintenanceProcedures {
		if maintenanceProcedure.PublicEventId == "" {
			return fmt.Errorf("empty msg-event-id for maintenance-procedure in %v", maintenanceProcedure.BpmnId)
		}
		if ok := knownEventIds[maintenanceProcedure.PublicEventId]; ok {
			return fmt.Errorf("reuse of %v as msg-event-id for maintenance-procedure in %v", maintenanceProcedure.PublicEventId, maintenanceProcedure.BpmnId)
		}
		knownEventIds[maintenanceProcedure.PublicEventId] = true

		knownParams := map[string]bool{}
		for _, param := range maintenanceProcedure.ParameterDescriptions {
			if ok := knownInitParams[param.Id]; ok {
				return fmt.Errorf("reuse of init-param %v as param-id in maintenance-procedure %v (%v)", param.Id, maintenanceProcedure.PublicEventId, maintenanceProcedure.BpmnId)
			}
			if ok := knownParams[param.Id]; ok {
				return fmt.Errorf("reuse of %v as param-id in maintenance-procedure %v (%v)", param.Id, maintenanceProcedure.PublicEventId, maintenanceProcedure.BpmnId)
			}
			knownParams[param.Id] = true
		}
	}
	return nil
}
