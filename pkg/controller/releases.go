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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/permissions"
	"github.com/beevik/etree"
	"log"
	"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"
)

//---------- API -------------

func (this *Controller) CreateRelease(token auth.Token, element model.SmartServiceRelease) (result model.SmartServiceRelease, err error, code int) {
	if this.releasesProducer == nil {
		return result, errors.New("edit is disabled"), http.StatusInternalServerError
	}
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

	if element.Id == "" {
		element.Id = this.GetNewId()
	}

	parsedInfo, err := this.parseDesignXmlForReleaseInfo(token, design.BpmnXml)
	if err != nil {
		return result, fmt.Errorf("unable to parse design xml for release: %w", err), http.StatusBadRequest
	}
	err = this.validateParsedReleaseInfos(parsedInfo)
	if err != nil {
		return result, err, http.StatusBadRequest
	}

	err = this.publishReleaseUpdate(token.GetUserId(), model.SmartServiceReleaseExtended{
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

func (this *Controller) publishReleaseUpdate(owner string, release model.SmartServiceReleaseExtended) (err error) {
	msg, err := json.Marshal(ReleaseCommand{
		Command: "PUT",
		Id:      release.Id,
		Owner:   owner,
		Release: &release,
	})
	if err != nil {
		return err
	}

	return this.releasesProducer.Produce(release.DesignId+"/"+release.Id, msg)
}

func (this *Controller) GetRelease(token auth.Token, id string) (result model.SmartServiceRelease, err error, code int) {
	access, err := this.permissions.CheckAccess(token, this.config.KafkaSmartServiceReleaseTopic, id, "r")
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !access {
		return result, errors.New("access denied"), http.StatusForbidden
	}
	var extended model.SmartServiceReleaseExtended
	extended, err, code = this.db.GetRelease(id)
	return extended.SmartServiceRelease, err, code
}

type ReleasePermissionsWrapper struct {
	Id          string          `json:"id"`
	Shared      bool            `json:"shared"`
	Permissions map[string]bool `json:"permissions"`
	DesignId    string          `json:"design_id"`
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
	access, err := this.permissions.CheckAccess(token, this.config.KafkaSmartServiceReleaseTopic, id, "r")
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !access {
		return result, errors.New("access denied"), http.StatusForbidden
	}
	return this.db.GetRelease(id)
}

func (this *Controller) ListExtendedReleases(token auth.Token, query model.ReleaseQueryOptions) (result []model.SmartServiceReleaseExtended, err error, code int) {
	permWrapper := []ReleasePermissionsWrapper{}
	var filter *permissions.Selection
	if query.Latest {
		filter = &permissions.Selection{
			Condition: permissions.ConditionConfig{
				Feature:   "features.new_release_id",
				Operation: permissions.QueryEqualOperation,
				Value:     "",
			},
		}
	}
	err, _ = this.permissions.Query(token.Jwt(), permissions.QueryMessage{
		Resource: this.config.KafkaSmartServiceReleaseTopic,
		Find: &permissions.QueryFind{
			QueryListCommons: permissions.QueryListCommons{
				Limit:    query.Limit,
				Offset:   query.Offset,
				Rights:   query.Rights,
				SortBy:   query.GetSortField(),
				SortDesc: !query.GetSortAsc(),
			},
			Filter: filter,
			Search: query.Search,
		},
	}, &permWrapper)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	idList := []string{}
	for _, id := range permWrapper {
		idList = append(idList, id.Id)
	}
	temp, err := this.db.GetReleaseListByIds(idList, query.GetSort())
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	permissionsInfoIndex := map[string]model.PermissionsInfo{}
	for _, perm := range permWrapper {
		permissionsInfoIndex[perm.Id] = model.PermissionsInfo{
			Shared:      perm.Shared,
			Permissions: perm.Permissions,
		}
	}
	for _, release := range temp {
		release.PermissionsInfo = permissionsInfoIndex[release.Id]
		result = append(result, release)
	}
	return result, nil, http.StatusOK
}

func (this *Controller) DeleteRelease(token auth.Token, releaseId string) (error, int) {
	if this.releasesProducer == nil {
		return errors.New("edit is disabled"), http.StatusInternalServerError
	}
	access, err := this.permissions.CheckAccess(token, this.config.KafkaSmartServiceReleaseTopic, releaseId, "a")
	if err != nil {
		return err, http.StatusInternalServerError
	}
	if !access {
		return errors.New("access denied"), http.StatusForbidden
	}

	//find design-id to create correct kafka-key
	wrapper := []ReleasePermissionsWrapper{}
	oldReleas, err, code := this.db.GetRelease(releaseId) //try database
	if err != nil && code == http.StatusNotFound {
		//if not found in the database: try permissions-search
		err, code = this.permissions.Query(token.Jwt(), permissions.QueryMessage{
			Resource: this.config.KafkaSmartServiceReleaseTopic,
			ListIds: &permissions.QueryListIds{
				QueryListCommons: permissions.QueryListCommons{
					Limit:  1,
					Rights: "a",
				},
				Ids: []string{releaseId},
			},
		}, &wrapper)
		if err != nil {
			return err, code
		}
		if len(wrapper) == 0 {
			return fmt.Errorf("unknown release id: %v", releaseId), http.StatusNotFound
		}
		oldReleas.DesignId = wrapper[0].DesignId
		err = nil
	}
	key := oldReleas.DesignId + "/" + releaseId

	msg, err := json.Marshal(ReleaseCommand{
		Command: "DELETE",
		Id:      releaseId,
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	err = this.releasesProducer.Produce(key, msg)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Controller) GetReleaseParameter(token auth.Token, id string) (result []model.SmartServiceExtendedParameter, err error, code int) {
	access, err := this.permissions.CheckAccess(token, this.config.KafkaSmartServiceReleaseTopic, id, "x")
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !access {
		return result, errors.New("access denied"), http.StatusForbidden
	}
	return this.GetReleaseParameterWithoutAuthCheck(token, id)
}

func (this *Controller) GetReleaseParameterWithoutAuthCheck(token auth.Token, id string) (result []model.SmartServiceExtendedParameter, err error, code int) {
	release, err, code := this.db.GetRelease(id)
	if err != nil {
		return result, err, code
	}
	for _, paramDesc := range release.ParsedInfo.ParameterDescriptions {
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

		//set default value to nil if characteristic is requested
		if param.CharacteristicId != nil && *param.CharacteristicId != "" {
			param.DefaultValue = nil
		}

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

//---------- CQRS -------------

type ReleaseCommand struct {
	Command string                             `json:"command"`
	Id      string                             `json:"id"`
	Owner   string                             `json:"owner"`
	Release *model.SmartServiceReleaseExtended `json:"release"`
}

func (this *Controller) HandleReleaseMessage(delivery []byte) error {
	release := ReleaseCommand{}
	err := json.Unmarshal(delivery, &release)
	if err != nil {
		log.Println("ERROR: consumed invalid message --> ignore", err)
		debug.PrintStack()
		return err
	}
	return this.HandleRelease(release)
}

func (this *Controller) HandleRelease(cmd ReleaseCommand) (err error) {
	switch cmd.Command {
	case "PUT":
		if cmd.Release == nil {
			log.Println("WARNING: missing release in release put command", cmd)
			return nil
		}
		err = this.HandleReleaseSave(cmd.Owner, *cmd.Release)
		if err != nil {
			return err
		}
		return nil
	case "RIGHTS":
		return nil
	case "DELETE":
		err = this.HandleReleaseDelete(cmd.Owner, cmd.Id)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("unable to handle command: " + cmd.Command)
	}
}

func (this *Controller) HandleReleaseSave(owner string, release model.SmartServiceReleaseExtended) (err error) {
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
		err = this.copyRightsOfRelease(owner, oldReleases[len(oldReleases)-1], release)
		if err != nil {
			return err
		}
	}

	err, _ = this.db.SetRelease(release)
	if err != nil {
		return err
	}
	err, isInvalidCamundaDeployment := this.camunda.DeployRelease(owner, release)
	if err != nil {
		errOnErrNotification := this.db.SetReleaseError(release.Id, err.Error())
		if isInvalidCamundaDeployment {
			return errOnErrNotification
		} else {
			return err
		}
	}
	err = this.db.SetReleaseError(release.Id, "")
	if err != nil {
		return err
	}
	for _, old := range oldReleases {
		if old.CreatedAt < release.CreatedAt { //"if" to prevent race from  HandleReleaseDelete() to recreate deleted release
			old.NewReleaseId = release.Id
			err = this.publishReleaseUpdate(owner, old)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (this *Controller) HandleReleaseDelete(userId string, id string) error {
	//remove release from camunda
	err := this.camunda.RemoveRelease(id)
	if err != nil {
		return err
	}

	//update NewReleaseId on other releases if this release is the newest one
	currentRelease, err, code := this.db.GetRelease(id)
	if err != nil && code != http.StatusNotFound {
		return err
	}
	if err == nil && currentRelease.NewReleaseId == "" {
		oldReleases, err := this.db.GetReleasesByDesignId(currentRelease.DesignId)
		if err != nil {
			return err
		}
		sort.Slice(oldReleases, func(i, j int) bool {
			return oldReleases[i].CreatedAt > oldReleases[i].CreatedAt
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
			err = this.publishReleaseUpdate(userId, youngestRelease)
			if err != nil {
				return err
			}
		}
		//other releases will be updated on update handling of youngestRelease because NewReleaseId == ""
		//there is a race between the deletion of this release from the database and the update of releases that are not youngestRelease in HandleReleaseSave()
		//but the retroactive create/uptdate of the releaste that is meant to be deleted is prevented by "if old.CreatedAt < release.CreatedAt {" in HandleReleaseSave()
	}

	//delete release from db
	err, _ = this.db.DeleteRelease(id)
	return err
}

func (this *Controller) copyRightsOfRelease(owner string, oldRelease model.SmartServiceReleaseExtended, newRelease model.SmartServiceReleaseExtended) error {
	token, err := this.adminAccess.EnsureAccess(this.config)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return err
	}
	rights, err, code := this.permissions.GetResourceRights(token, this.config.KafkaSmartServiceReleaseTopic, oldRelease.Id)
	if err != nil {
		log.Println("ERROR:", code, err)
		debug.PrintStack()
		return err
	}
	rights.UserRights[owner] = permissions.Right{
		Read:         true,
		Write:        true,
		Execute:      true,
		Administrate: true,
	}
	//same prefix as release PUT/DELETE to ensure same partition (preserved order when consuming)
	//butt different suffix to ensure separate compaction
	kafkaKey := newRelease.DesignId + "/" + newRelease.Id + "_rights"
	return this.permissions.SetResourceRights(token, this.config.KafkaSmartServiceReleaseTopic, newRelease.Id, rights, kafkaKey)
}

//------------ Parsing ----------------

func (this *Controller) parseDesignXmlForReleaseInfo(token auth.Token, xml string) (result model.SmartServiceReleaseInfo, err error) {
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
	for _, formField := range doc.FindElements("//camunda:formField") {
		id := formField.SelectAttrValue("id", "")
		if id == "" {
			return result, errors.New("missing id in camunda:formField")
		}
		label := formField.SelectAttrValue("label", id)
		if label == "" {
			label = id
		}
		fieldType := formField.SelectAttrValue("type", "")
		if id == "" {
			return result, errors.New("missing type in camunda:formField")
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
			param.Characteristic, err = this.GetCharacteristic(token.Jwt(), chId)
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
		result.ParameterDescriptions = append(result.ParameterDescriptions, param)
	}
	return result, nil
}

func (this *Controller) validateParsedReleaseInfos(info model.SmartServiceReleaseInfo) error {
	for _, param := range info.ParameterDescriptions {
		if param.AutoSelectAll && !param.Multiple {
			return fmt.Errorf("%v: parameter property \"auto_select_all\" may only be used in combination with  \"multiple\"", param.Id)
		}
		if param.IotDescription != nil {
			if param.IotDescription.NeedsSameEntityIdInParameter != "" {
				if !ListContains(info.ParameterDescriptions, func(p model.ParameterDescription) bool {
					return p.Id == param.IotDescription.NeedsSameEntityIdInParameter
				}) {
					return fmt.Errorf("%v: parameter property \"entity_only\" references unknown parameter \"%v\"", param.Id, param.IotDescription.NeedsSameEntityIdInParameter)
				}
			}
		}
	}
	return nil
}
