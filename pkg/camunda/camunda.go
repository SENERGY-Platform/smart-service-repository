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

package camunda

import (
	"errors"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"log"
	"net/url"
	"strings"
)

type Camunda struct {
	config configuration.Config
}

func New(config configuration.Config) *Camunda {
	return &Camunda{
		config: config,
	}
}

func idToCNName(id string) string {
	result := strings.ReplaceAll(id, "-", "_")
	if !strings.HasPrefix(result, "id_") {
		result = "id_" + result
	}
	return result
}

func (this *Camunda) filterUrlFromErr(in error) (out error) {
	if this.config.Debug {
		log.Println("DEBUG: transform error message:", in)
	}
	text := in.Error()
	text = strings.ReplaceAll(text, this.config.CamundaUrl, "http://camunda:8080")
	parsed, err := url.Parse(this.config.CamundaUrl)
	if err == nil {
		text = strings.ReplaceAll(text, parsed.Hostname(), "camunda")
		text = strings.ReplaceAll(text, parsed.User.Username()+":***@camunda", "camunda")
	}
	return errors.New(text)
}
