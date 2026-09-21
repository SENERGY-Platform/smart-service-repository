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
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"reflect"
	"testing"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
)

// A design names the aspects of an iot parameter in the criteria property of a form field,
// either as the deprecated single aspect_id or as the aspect_ids list. Both spellings reach
// the parsed release exactly as they were written; device-selection folds one into the other.
func TestParseCriteriaAspectIds(t *testing.T) {
	aspectId := func(value string) *string { return &value }

	tests := []struct {
		name     string
		property string
		value    string
		expected []model.Criteria
	}{
		{
			name:     "criteria property with an aspect list",
			property: "criteria",
			value:    `{"function_id":"fid","aspect_ids":["aid1","aid2"]}`,
			expected: []model.Criteria{{FunctionId: aspectId("fid"), AspectIds: []string{"aid1", "aid2"}}},
		},
		{
			name:     "criteria property with the deprecated aspect id",
			property: "criteria",
			value:    `{"function_id":"fid","aspect_id":"aid"}`,
			expected: []model.Criteria{{FunctionId: aspectId("fid"), AspectId: aspectId("aid")}},
		},
		{
			name:     "criteria property with both spellings",
			property: "criteria",
			value:    `{"aspect_id":"aid1","aspect_ids":["aid2"]}`,
			expected: []model.Criteria{{AspectId: aspectId("aid1"), AspectIds: []string{"aid2"}}},
		},
		{
			name:     "criteria_list property with an aspect list per entry",
			property: "criteria_list",
			value:    `[{"function_id":"f1","aspect_ids":["aid1","aid2"]},{"function_id":"f2","aspect_ids":["aid3"]}]`,
			expected: []model.Criteria{
				{FunctionId: aspectId("f1"), AspectIds: []string{"aid1", "aid2"}},
				{FunctionId: aspectId("f2"), AspectIds: []string{"aid3"}},
			},
		},
		{
			name:     "criteria_list property mixing both spellings",
			property: "criteria_list",
			value:    `[{"function_id":"f1","aspect_id":"aid1"},{"function_id":"f2","aspect_ids":["aid2"]}]`,
			expected: []model.Criteria{
				{FunctionId: aspectId("f1"), AspectId: aspectId("aid1")},
				{FunctionId: aspectId("f2"), AspectIds: []string{"aid2"}},
			},
		},
		{
			name:     "criteria property without an aspect leaves the list unset",
			property: "criteria",
			value:    `{"function_id":"fid"}`,
			expected: []model.Criteria{{FunctionId: aspectId("fid")}},
		},
	}

	ctrl := &Controller{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info, err := ctrl.parseDesignXmlForReleaseInfo(context.Background(), auth.Token{}, bpmnWithCriteriaProperty(test.property, test.value), model.SmartServiceRelease{})
			if err != nil {
				t.Error(err)
				return
			}
			if len(info.ParameterDescriptions) != 1 {
				t.Error(len(info.ParameterDescriptions))
				return
			}
			iot := info.ParameterDescriptions[0].IotDescription
			if iot == nil {
				t.Error("missing iot description")
				return
			}
			if !reflect.DeepEqual(iot.Criteria, test.expected) {
				t.Error(iot.Criteria, test.expected)
			}
		})
	}
}

// bpmnWithCriteriaProperty builds the smallest design that carries one iot parameter, with the
// given criteria property set to value. The value is json inside an xml attribute, so it has
// to be escaped the way a modeler would write it.
func bpmnWithCriteriaProperty(property string, value string) string {
	escaped := &bytes.Buffer{}
	err := xml.EscapeText(escaped, []byte(value))
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
	<bpmn:process id="criteria_aspect_ids_test" isExecutable="true">
		<bpmn:startEvent id="StartEvent_1">
			<bpmn:extensionElements>
				<camunda:formData>
					<camunda:formField id="device" label="device" type="string">
						<camunda:properties>
							<camunda:property id="iot" value="device" />
							<camunda:property id="%s" value="%s" />
						</camunda:properties>
					</camunda:formField>
				</camunda:formData>
			</bpmn:extensionElements>
		</bpmn:startEvent>
	</bpmn:process>
</bpmn:definitions>`, property, escaped.String())
}
