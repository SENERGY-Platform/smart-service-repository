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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"net/http"
)

func (this *Controller) AddModule(token auth.Token, element model.SmartServiceModule) (result model.SmartServiceModule, err error, code int) {
	err, code = this.ValidateModule(token, element)
	if err != nil {
		return result, err, code
	}
	err, code = this.db.SetModule(element)
	if err != nil {
		return result, err, code
	}
	return this.db.GetModule(element.Id, token.GetUserId())
}

func (this *Controller) ListModules(token auth.Token, query model.ModuleQueryOptions) ([]model.SmartServiceModule, error, int) {
	return this.db.ListModules(token.GetUserId(), query)
}

func (this *Controller) ValidateModule(token auth.Token, element model.SmartServiceModule) (error, int) {
	if element.Id == "" {
		return errors.New("missing id"), http.StatusBadRequest
	}
	if element.UserId == "" {
		return errors.New("missing user id"), http.StatusBadRequest
	}
	if element.InstanceId == "" {
		return errors.New("missing instance id"), http.StatusBadRequest
	}
	instance, err, code := this.db.GetInstance(element.Id, token.GetUserId())
	if err != nil {
		if code == http.StatusNotFound {
			code = http.StatusBadRequest
		}
		return fmt.Errorf("referenced smart service instance not found: %w", err), code
	}
	if instance.UserId != element.UserId {
		return errors.New("referenced smart service instance is owned by a different user"), http.StatusForbidden
	}
	return nil, http.StatusOK
}
