/*
 * Copyright (c) 2026 InfAI (CC SES)
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
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TestStartSendsBaggage ensures that the smart-service-instance-id, which CreateInstance puts
// into the context, reaches camunda as a baggage header and can be passed on from there.
func TestStartSendsBaggage(t *testing.T) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	mux := sync.Mutex{}
	receivedBaggage := map[string]string{}
	camundaMock := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mux.Lock()
		defer mux.Unlock()
		receivedBaggage[request.URL.Path] = request.Header.Get("baggage")
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer camundaMock.Close()

	config := configuration.Config{CamundaUrl: camundaMock.URL}

	ctx, err := tracing.AddToBaggage(context.Background(), tracing.BaggageKeyInstanceId, "instance-id-1")
	if err != nil {
		t.Error(err)
		return
	}

	err = New(config).Start(ctx, model.SmartServiceInstance{
		Id:        "instance-id-1",
		ReleaseId: "release-id-1",
	})
	if err != nil {
		t.Error(err)
		return
	}

	mux.Lock()
	defer mux.Unlock()
	if len(receivedBaggage) == 0 {
		t.Error("camunda has not been called")
		return
	}
	for path, value := range receivedBaggage {
		if !strings.Contains(value, tracing.BaggageKeyInstanceId+"=instance-id-1") {
			t.Errorf("missing instance id in baggage of %v: %#v", path, value)
		}
	}
}
