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
	endpoints = append(endpoints, &Designs{})
}

type Designs struct{}

// List godoc
// @Summary      returns a list of smart-service designs
// @Description  returns a list of smart-service designs
// @Tags         designs
// @Produce      json
// @Success      200 {array} model.SmartServiceDesign
// @Failure      500
// @Failure      401
// @Router       /designs [get]
func (this *Designs) List(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/designs", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode([]model.SmartServiceDesign{})
	})
}

// Get godoc
// @Summary      returns a smart-service designs
// @Description  returns a smart-service designs
// @Tags         designs
// @Produce      json
// @Param        id path string true "Design ID"
// @Success      200 {object} model.SmartServiceDesign
// @Failure      500
// @Failure      401
// @Router       /designs/{id} [get]
func (this *Designs) Get(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/designs/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
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

// Update godoc
// @Summary      updates a smart-service designs
// @Description  updates a smart-service designs
// @Tags         designs
// @Accept       json
// @Produce      json
// @Param        id path string true "Design ID"
// @Param        message body model.SmartServiceDesign true "SmartServiceDesign"
// @Success      200 {object} model.SmartServiceDesign
// @Failure      500
// @Failure      401
// @Router       /designs/{id} [put]
func (this *Designs) Update(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.PUT("/designs/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
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

// Create godoc
// @Summary      creates a smart-service designs
// @Description  creates a smart-service designs
// @Tags         designs
// @Accept       json
// @Produce      json
// @Param        id path string true "Design ID"
// @Param        message body model.SmartServiceDesign true "SmartServiceDesign"
// @Success      200 {object} model.SmartServiceDesign
// @Failure      500
// @Failure      401
// @Router       /designs [post]
func (this *Designs) Create(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/designs", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
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

// Delete godoc
// @Summary      removes a smart-service designs
// @Description  removes a smart-service designs
// @Tags         designs
// @Param        id path string true "Design ID"
// @Success      200
// @Failure      500
// @Failure      401
// @Router       /designs/{id} [delete]
func (this *Designs) Delete(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/designs/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
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
