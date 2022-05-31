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
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"net/http"
)

func (this *Controller) ListDesigns(token auth.Token, query model.DesignQueryOptions) ([]model.SmartServiceDesign, error, int) {
	return this.db.ListDesigns(token.GetUserId(), query)
}

func (this *Controller) GetDesign(token auth.Token, id string) (result model.SmartServiceDesign, err error, code int) {
	return this.db.GetDesign(id, token.GetUserId())
}

func (this *Controller) SetDesign(token auth.Token, element model.SmartServiceDesign) (result model.SmartServiceDesign, err error, code int) {
	err, code = this.ValidateDesign(token, element)
	if err != nil {
		return result, err, code
	}
	err, code = this.db.SetDesign(element)
	if err != nil {
		return result, err, code
	}
	return this.db.GetDesign(element.Id, token.GetUserId())
}

func (this *Controller) DeleteDesign(token auth.Token, id string) (error, int) {
	return this.db.DeleteDesign(id, token.GetUserId())
}

func (this *Controller) ValidateDesign(token auth.Token, element model.SmartServiceDesign) (err error, code int) {
	if element.Id == "" {
		return errors.New("missing id"), http.StatusBadRequest
	}
	if element.UserId == "" {
		return errors.New("missing user id"), http.StatusBadRequest
	}
	return nil, http.StatusOK
}
