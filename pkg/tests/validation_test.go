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

package tests

import (
	"github.com/SENERGY-Platform/smart-service-repository/pkg/controller"
	"os"
	"strings"
	"testing"
)

func TestValidation(t *testing.T) {
	dir := "./resources/validation/"
	infos, err := os.ReadDir(dir)
	if err != nil {
		t.Error(err)
		return
	}
	for _, info := range infos {
		name := info.Name()
		if !info.IsDir() {
			t.Run(name, func(t *testing.T) {
				runValidationTest(t, dir+name, strings.HasPrefix(name, "valid_"))
			})
		}
	}
}

func runValidationTest(t *testing.T, location string, expectValid bool) {
	fileContent, err := os.ReadFile(location)
	if err != nil {
		t.Error(err)
		return
	}
	err = controller.ValidateDesign(string(fileContent))
	if expectValid && err != nil {
		t.Error(err)
	}
	if !expectValid && err == nil {
		t.Error("expected error for", location)
	}
}
