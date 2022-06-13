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

	parsedInfo, err := this.parseDesignXmlForReleaseInfo(design.BpmnXml)
	if err != nil {
		return result, fmt.Errorf("unable to parse design xml for release: %w", err), http.StatusBadRequest
	}

	msg, err := json.Marshal(ReleaseCommand{
		Command: "PUT",
		Id:      element.Id,
		Owner:   token.GetUserId(),
		Release: &model.SmartServiceReleaseExtended{
			SmartServiceRelease: element,
			BpmnXml:             design.BpmnXml,
			SvgXml:              design.SvgXml,
			ParsedInfo:          parsedInfo,
		},
	})
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	err = this.releasesProducer.Produce(element.Id, msg)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	return element, nil, http.StatusOK
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

type IdWrapper struct {
	Id string `json:"id"`
}

func (this *Controller) ListReleases(token auth.Token, query model.ReleaseQueryOptions) (result []model.SmartServiceRelease, err error, code int) {
	idWrapperList := []IdWrapper{}
	err, _ = this.permissions.Query(token.Jwt(), permissions.QueryMessage{
		Resource: this.config.KafkaSmartServiceReleaseTopic,
		Find: &permissions.QueryFind{
			QueryListCommons: permissions.QueryListCommons{
				Limit:    query.Limit,
				Offset:   query.Offset,
				Rights:   "r",
				SortBy:   query.GetSortField(),
				SortDesc: !query.GetSortAsc(),
			},
		},
	}, &idWrapperList)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	idList := []string{}
	for _, id := range idWrapperList {
		idList = append(idList, id.Id)
	}
	temp, err := this.db.GetReleaseListByIds(idList, query.GetSort())
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	for _, release := range temp {
		result = append(result, release.SmartServiceRelease)
	}
	return result, nil, http.StatusOK
}

func (this *Controller) DeleteRelease(token auth.Token, id string) (error, int) {
	if this.releasesProducer == nil {
		return errors.New("edit is disabled"), http.StatusInternalServerError
	}
	access, err := this.permissions.CheckAccess(token, this.config.KafkaSmartServiceReleaseTopic, id, "a")
	if err != nil {
		return err, http.StatusInternalServerError
	}
	if !access {
		return errors.New("access denied"), http.StatusForbidden
	}
	msg, err := json.Marshal(ReleaseCommand{
		Command: "DELETE",
		Id:      id,
	})
	if err != nil {
		return err, http.StatusInternalServerError
	}
	err = this.releasesProducer.Produce(id, msg)
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
	release, err, code := this.db.GetRelease(id)
	if err != nil {
		return result, err, code
	}
	for _, paramDesc := range release.ParsedInfo.ParameterDescriptions {
		param := model.SmartServiceExtendedParameter{
			SmartServiceParameter: model.SmartServiceParameter{
				Id:    paramDesc.Id,
				Value: paramDesc.DefaultValue,
			},
			Label:        paramDesc.Label,
			Description:  paramDesc.Description,
			DefaultValue: paramDesc.DefaultValue,
			Type:         getSchemaOrgType(paramDesc.Type),
			Multiple:     paramDesc.Multiple,
			Order:        paramDesc.Order,
		}
		param.Options, err, code = this.getParamOptions(token, paramDesc)
		if err != nil {
			return result, err, code
		}
		result = append(result, param)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Order < result[j].Order
	})
	return result, nil, http.StatusOK
}

func getSchemaOrgType(t string) model.Type {
	switch t {
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
	case "DELETE":
		err = this.HandleReleaseDelete(cmd.Id)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("unable to handle command: " + cmd.Command)
	}
}

func (this *Controller) HandleReleaseSave(owner string, release model.SmartServiceReleaseExtended) error {
	err, _ := this.db.SetRelease(release)
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
	return this.db.SetReleaseError(release.Id, "")
}

func (this *Controller) HandleReleaseDelete(id string) error {
	err := this.camunda.RemoveRelease(id)
	if err != nil {
		return err
	}
	err, _ = this.db.DeleteRelease(id)
	return err
}

//------------ Parsing ----------------

func (this *Controller) parseDesignXmlForReleaseInfo(xml string) (result model.SmartServiceReleaseInfo, err error) {
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
		}
		if order, ok := properties["order"]; ok {
			param.Order, err = strconv.Atoi(order)
			if err != nil {
				return result, fmt.Errorf("invalid order property for formField %v: %w", id, err)
			}
		}
		if options, ok := properties["options"]; ok {
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
		if iot, ok := properties["iot"]; ok {
			if _, containsOptions := properties["options"]; containsOptions {
				return result, fmt.Errorf("invalid options/iot property for formField %v: %v", id, "iot and options are mutual exclusive")
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
