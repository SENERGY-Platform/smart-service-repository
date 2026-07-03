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
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/beevik/etree"
)

func (this *Controller) parseDesignXmlForReleaseInfo(token auth.Token, xml string, element model.SmartServiceRelease) (result model.SmartServiceReleaseInfo, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			err = fmt.Errorf("panic in parseDesignXmlForReleaseInfo: %v", r)
			this.config.GetLogger().Error("Recovered Error", "error", r, "stack", string(debug.Stack()))
		}
	}()
	doc := etree.NewDocument()
	err = doc.ReadFromString(xml)
	if err != nil {
		return result, err
	}
	result.ModuleInfo, err = this.parseReleaseModuleInfo(doc)
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

const AnalyticsTopic = "analytics"
const AnalyticsParamPrefix = "analytics."

func (this *Controller) parseReleaseModuleInfo(doc *etree.Document) (result model.ReleaseModuleInfo, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			err = fmt.Errorf("panic in parseReleaseModuleInfo: %v", r)
			this.config.GetLogger().Error("Recovered Error", "error", r, "stack", string(debug.Stack()))
		}
	}()
	result.Analytics = []model.AnalyticsReleaseModuleInfo{}
	for _, element := range doc.FindElements("//bpmn:serviceTask[@camunda:topic='" + AnalyticsTopic + "']") {
		analyticsInfo, err := this.parseAnalyticsReleaseModuleInfo(element)
		if err != nil {
			return result, err
		}
		if analyticsInfo.FlowId != "" {
			result.Analytics = append(result.Analytics, analyticsInfo)
		}
	}
	return result, nil
}

func (this *Controller) parseAnalyticsReleaseModuleInfo(element *etree.Element) (result model.AnalyticsReleaseModuleInfo, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			err = fmt.Errorf("panic in parseAnalyticsReleaseModuleInfo: %v", r)
			this.config.GetLogger().Error("Recovered Error", "error", r, "stack", string(debug.Stack()))
		}
	}()
	for _, param := range element.FindElements(".//camunda:inputParameter") {
		paramName := param.SelectAttrValue("name", "")
		paramValue := param.Text()
		switch paramName {
		case AnalyticsParamPrefix + "flow_id":
			result.FlowId = paramValue
		case AnalyticsParamPrefix + "name":
			result.Name = paramValue
		case AnalyticsParamPrefix + "desc":
			result.Desc = paramValue
		}
	}
	return result, nil
}

func (this *Controller) ensureValidReleaseModuleInfo(element model.SmartServiceReleaseExtended) (result model.SmartServiceReleaseExtended, err error) {
	if element.ParsedInfo.ModuleInfo.Analytics != nil {
		return element, nil
	}
	result = element
	doc := etree.NewDocument()
	err = doc.ReadFromString(element.BpmnXml)
	if err != nil {
		return result, fmt.Errorf("unable to parse bpmn xml for ensureValidReleaseModuleInfo: %w", err)
	}
	result.ParsedInfo.ModuleInfo, err = this.parseReleaseModuleInfo(doc)
	if err != nil {
		return result, fmt.Errorf("unable to parse release module info for ensureValidReleaseModuleInfo: %w", err)
	}
	return result, nil
}
