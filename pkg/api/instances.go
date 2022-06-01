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

package api

import (
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
)

func init() {
	endpoints = append(endpoints, &Instances{})
}

type Instances struct{}

// List godoc
// @Summary      returns a list of smart-service instances
// @Description  returns a list of smart-service instances
// @Tags         instances
// @Produce      json
// @Success      200 {array}  model.SmartServiceInstance
// @Failure      500
// @Failure      401
// @Router       /instances [get]
func (this *Instances) List(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/instances", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode([]model.SmartServiceInstance{})
	})
}

// Get godoc
// @Summary      returns a smart-service instance
// @Description  returns a smart-service instance
// @Tags         instances
// @Produce      json
// @Param        id path string true "Instance ID"
// @Success      200 {object}  model.SmartServiceInstance
// @Failure      500
// @Failure      401
// @Router       /instances/{id} [get]
func (this *Instances) Get(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/instances/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceInstance{})
	})
}

// Update godoc
// @Summary      updates smart-service instance parameter
// @Description  updates smart-service instance parameter
// @Tags         instances, parameter
// @Accept       json
// @Produce      json
// @Param        id path string true "Instance ID"
// @Param        message body model.SmartServiceParameters true "SmartServiceParameter"
// @Success      200 {object}  model.SmartServiceInstance
// @Failure      500
// @Failure      401
// @Router       /instances/{id}/parameter [patch]
func (this *Instances) Update(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.PATCH("/instances/:id/parameter", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceInstance{})
	})
}

// Delete godoc
// @Summary      removes a smart-service instance with all modules
// @Description  removes a smart-service instance with all modules
// @Tags         instances
// @Param        id path string true "Instance ID"
// @Success      200
// @Failure      500
// @Failure      401
// @Router       /instances/{id} [delete]
func (this *Instances) Delete(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.DELETE("/instances/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceDesign{})
	})
}
