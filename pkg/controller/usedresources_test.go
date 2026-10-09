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
	"reflect"
	"testing"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/resources"
)

func usedResourcesTestBpmn(serviceTasks string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
    <bpmn:process id="Process_1" isExecutable="true">
        <bpmn:startEvent id="StartEvent_1"/>
` + serviceTasks + `
    </bpmn:process>
</bpmn:definitions>`
}

func serviceTask(id string, topic string, params string) string {
	return `<bpmn:serviceTask id="` + id + `" camunda:type="external" camunda:topic="` + topic + `">
            <bpmn:extensionElements><camunda:inputOutput>` + params + `</camunda:inputOutput></bpmn:extensionElements>
        </bpmn:serviceTask>`
}

func TestParseReleaseUsedResources(t *testing.T) {
	cases := []struct {
		name     string
		tasks    string
		expected model.ReleaseUsedResources
	}{
		{
			name:     "no service tasks",
			tasks:    "",
			expected: model.ReleaseUsedResources{ProcessModels: []string{}, Flows: []string{}, ImportTypes: []string{}},
		},
		{
			name: "process models: literal ids are trimmed, deduplicated and sorted; expressions and scripts are skipped",
			tasks: serviceTask("t1", "process_deployment", `<camunda:inputParameter name="process_deployment.process_model_id">
                        pm-b
                    </camunda:inputParameter>`) +
				serviceTask("t2", "process_deployment", `<camunda:inputParameter name="process_deployment.process_model_id">pm-a</camunda:inputParameter>`) +
				serviceTask("t3", "process_deployment", `<camunda:inputParameter name="process_deployment.process_model_id">pm-b</camunda:inputParameter>`) +
				serviceTask("t4", "process_deployment", `<camunda:inputParameter name="process_deployment.process_model_id">${model_id}</camunda:inputParameter>`) +
				serviceTask("t5", "process_deployment", `<camunda:inputParameter name="process_deployment.process_model_id">prefix-#{model_id}</camunda:inputParameter>`) +
				serviceTask("t6", "process_deployment", `<camunda:inputParameter name="process_deployment.process_model_id"><camunda:script scriptFormat="JavaScript">"pm-script"</camunda:script></camunda:inputParameter>`) +
				serviceTask("t7", "process_deployment", `<camunda:inputParameter name="process_deployment.process_model_id"></camunda:inputParameter>`) +
				serviceTask("t8", "process_deployment", `<camunda:inputParameter name="process_deployment.process_model_id">pm-mixed<camunda:list><camunda:value>pm-list</camunda:value></camunda:list></camunda:inputParameter>`),
			expected: model.ReleaseUsedResources{ProcessModels: []string{"pm-a", "pm-b"}, Flows: []string{}, ImportTypes: []string{}},
		},
		{
			name: "parameters count only on service tasks of their topic and with the prefixed name",
			tasks: serviceTask("t1", "analytics", `<camunda:inputParameter name="process_deployment.process_model_id">pm-wrong-topic</camunda:inputParameter>`) +
				serviceTask("t2", "process_deployment", `<camunda:inputParameter name="process_model_id">pm-unprefixed</camunda:inputParameter>`) +
				serviceTask("t3", "process_deployment", `<camunda:inputParameter name="analytics.flow_id">flow-wrong-topic</camunda:inputParameter>`),
			expected: model.ReleaseUsedResources{ProcessModels: []string{}, Flows: []string{}, ImportTypes: []string{}},
		},
		{
			name: "flows: literal ids count, expressions are skipped",
			tasks: serviceTask("t1", "analytics", `<camunda:inputParameter name="analytics.flow_id">flow-2</camunda:inputParameter><camunda:inputParameter name="analytics.name">n</camunda:inputParameter>`) +
				serviceTask("t2", "analytics", `<camunda:inputParameter name="analytics.flow_id"> flow-1 </camunda:inputParameter>`) +
				serviceTask("t3", "analytics", `<camunda:inputParameter name="analytics.flow_id">${flow}</camunda:inputParameter>`),
			expected: model.ReleaseUsedResources{ProcessModels: []string{}, Flows: []string{"flow-1", "flow-2"}, ImportTypes: []string{}},
		},
		{
			name: "import types: id from the json request, with xml entities and expressions in other fields",
			tasks: serviceTask("t1", "import", `<camunda:inputParameter name="import.request">{&#34;import_type_id&#34;: &#34;it-1&#34;, &#34;name&#34;: &#34;${name}&#34;}</camunda:inputParameter>`) +
				serviceTask("t2", "import", `<camunda:inputParameter name="import.request">{"import_type_id": "${type}"}</camunda:inputParameter>`) +
				serviceTask("t3", "import", `<camunda:inputParameter name="import.request">{"import_type_id": 42}</camunda:inputParameter>`) +
				serviceTask("t4", "import", `<camunda:inputParameter name="import.request">{"name": "no type"}</camunda:inputParameter>`) +
				serviceTask("t5", "import", `<camunda:inputParameter name="import.request"><camunda:script scriptFormat="JavaScript">JSON.stringify({"import_type_id": "it-script"})</camunda:script></camunda:inputParameter>`),
			expected: model.ReleaseUsedResources{ProcessModels: []string{}, Flows: []string{}, ImportTypes: []string{"it-1"}},
		},
		{
			name: "import types: malformed requests yield the literal id if one is recognizable, else nothing",
			tasks: serviceTask("t1", "import", `<camunda:inputParameter name="import.request">{"import_type_id": "it-2", "limit": ${limit}}</camunda:inputParameter>`) +
				serviceTask("t2", "import", `<camunda:inputParameter name="import.request">{"import_type_id": "${type}", "limit": ${limit}}</camunda:inputParameter>`) +
				serviceTask("t3", "import", `<camunda:inputParameter name="import.request">not json at all</camunda:inputParameter>`) +
				serviceTask("t4", "import", `<camunda:inputParameter name="import.request">{"import_type_id": "it-3</camunda:inputParameter>`),
			expected: model.ReleaseUsedResources{ProcessModels: []string{}, Flows: []string{}, ImportTypes: []string{"it-2"}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := parseReleaseUsedResources(usedResourcesTestBpmn(c.tasks))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result, c.expected) {
				t.Errorf("\nexpected %#v\ngot      %#v", c.expected, result)
			}
		})
	}

	t.Run("existing example with v2 parameters", func(t *testing.T) {
		result, err := parseReleaseUsedResources(resources.AnalyticsExampleSmartService1)
		if err != nil {
			t.Fatal(err)
		}
		expected := model.ReleaseUsedResources{
			ProcessModels: []string{"2b93d779-7834-4aad-9d44-9723ae0953ca"},
			Flows:         []string{"62bd43205e353260a94ad6cd"},
			ImportTypes:   []string{},
		}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("\nexpected %#v\ngot      %#v", expected, result)
		}
	})

	t.Run("invalid xml", func(t *testing.T) {
		_, err := parseReleaseUsedResources("<bpmn:definitions")
		if err == nil {
			t.Error("expected error")
		}
	})
}
