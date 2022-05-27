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
	endpoints = append(endpoints, &Models{})
}

type Models struct{}

// List godoc
// @Summary      returns a list of smart-service models
// @Description  returns a list of smart-service models
// @Tags         models
// @Produce      json
// @Success      200 {array} model.SmartServiceModel
// @Failure      500
// @Failure      401
// @Router       /models [get]
func (this *Models) List(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/models", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode([]model.SmartServiceModel{})
	})
}

// Get godoc
// @Summary      returns a smart-service model
// @Description  returns a smart-service model
// @Tags         models
// @Produce      json
// @Param        id path string true "Model ID"
// @Success      200 {object} model.SmartServiceModel
// @Failure      500
// @Failure      401
// @Router       /models/{id} [get]
func (this *Models) Get(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/models/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceModel{})
	})
}

// Update godoc
// @Summary      updates a smart-service model
// @Description  updates a smart-service model
// @Tags         models
// @Accept       json
// @Produce      json
// @Param        id path string true "Model ID"
// @Param        message body model.SmartServiceModel true "SmartServiceModel"
// @Success      200 {object} model.SmartServiceModel
// @Failure      500
// @Failure      401
// @Router       /models/{id} [put]
func (this *Models) Update(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.PUT("/models/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceModel{})
	})
}

// Create godoc
// @Summary      creates a smart-service model
// @Description  creates a smart-service model
// @Tags         models
// @Accept       json
// @Produce      json
// @Param        id path string true "Model ID"
// @Param        message body model.SmartServiceModel true "SmartServiceModel"
// @Success      200 {object} model.SmartServiceModel
// @Failure      500
// @Failure      401
// @Router       /models [post]
func (this *Models) Create(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/models", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceModel{})
	})
}

// Delete godoc
// @Summary      removes a smart-service model
// @Description  removes a smart-service model
// @Tags         models
// @Param        id path string true "Model ID"
// @Success      200
// @Failure      500
// @Failure      401
// @Router       /models/{id} [delete]
func (this *Models) Delete(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/models/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceModel{})
	})
}
