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

package resources

import (
	_ "embed"
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"log"
	"runtime/debug"
)

//go:embed nameanddesc.bpmn
var NamedDescBpmn string

//go:embed nameanddesc.svg
var NamedDescSvg string

//go:embed params.bpmn
var ParamsBpmn string

//go:embed params.svg
var ParamsSvg string

//go:embed process_deployment.bpmn
var ProcessDeploymentBpmn string

//go:embed process_deployment.svg
var ProcessDeploymentSvg string

//go:embed complex_selection.bpmn
var ComplexSelectionBpmn string

//go:embed complex_selection.svg
var ComplexSelectionSvg string

//go:embed selections_response_1.json
var SelectionsResponse1 []byte
var SelectionsResponse1Obj []model.Selectable

//go:embed expected_params_1.json
var ExpectedParams1 []byte
var ExpectedParams1Obj []model.SmartServiceExtendedParameter

func init() {
	err := json.Unmarshal(SelectionsResponse1, &SelectionsResponse1Obj)
	if err != nil {
		debug.PrintStack()
		log.Println("ERROR:", err)
		panic(err)
	}
	err = json.Unmarshal(ExpectedParams1, &ExpectedParams1Obj)
	if err != nil {
		debug.PrintStack()
		log.Println("ERROR:", err)
		panic(err)
	}
}
