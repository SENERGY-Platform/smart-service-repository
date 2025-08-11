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
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/beevik/etree"
)

func ValidateDesign(config configuration.Config, xml string) (err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			config.GetLogger().Error("Recovered Error", "error", r, "stack", string(debug.Stack()))
			err = errors.New(fmt.Sprint("Recovered Error: ", r))
		}
	}()
	doc := etree.NewDocument()
	err = doc.ReadFromString(xml)
	if err != nil {
		return err
	}
	err = validateStartEvents(doc)
	if err != nil {
		return err
	}
	return nil
}

func validateStartEvents(doc *etree.Document) (err error) {
	startEvents := doc.FindElements("//bpmn:startEvent")
	hasDefaultStart := false
	for _, startEvent := range startEvents {
		msgEvent := startEvent.FindElement(".//bpmn:messageEventDefinition")
		if msgEvent != nil && msgEvent.SelectAttr("messageRef") == nil {
			return fmt.Errorf("missing message event name")
		} else if startEvent.FindElement(".//bpmn:conditionalEventDefinition") != nil {
			return fmt.Errorf("conditional start-events are not allowed")
		} else if startEvent.FindElement(".//bpmn:signalEventDefinition") != nil {
			return fmt.Errorf("signal start-events are not allowed")
		} else if startEvent.FindElement(".//bpmn:timerEventDefinition") != nil {
			return fmt.Errorf("time start-events are not allowed")
		} else {
			hasDefaultStart = true
		}
	}
	if !hasDefaultStart {
		return fmt.Errorf("missing default start-event")
	}
	return nil
}
