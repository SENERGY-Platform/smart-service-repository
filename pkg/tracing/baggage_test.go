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

package tracing

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/SENERGY-Platform/gin-middleware/otelx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
)

func TestAddToBaggage(t *testing.T) {
	ctx, err := AddToBaggage(context.Background(), BaggageKeyInstanceId, "instance-id-1")
	if err != nil {
		t.Error(err)
		return
	}
	value := baggage.FromContext(ctx).Member(BaggageKeyInstanceId).Value()
	if value != "instance-id-1" {
		t.Error(value)
		return
	}
}

func TestAddToBaggageKeepsExistingMembers(t *testing.T) {
	ctx, err := AddToBaggage(context.Background(), "user_id", "user-id-1")
	if err != nil {
		t.Error(err)
		return
	}
	ctx, err = AddToBaggage(ctx, BaggageKeyInstanceId, "instance-id-1")
	if err != nil {
		t.Error(err)
		return
	}
	if value := baggage.FromContext(ctx).Member("user_id").Value(); value != "user-id-1" {
		t.Error(value)
		return
	}
	if value := baggage.FromContext(ctx).Member(BaggageKeyInstanceId).Value(); value != "instance-id-1" {
		t.Error(value)
		return
	}
}

func TestAddToBaggageInvalidValue(t *testing.T) {
	_, err := AddToBaggage(context.Background(), BaggageKeyInstanceId, "invalid value with spaces")
	if err == nil {
		t.Error("expected error for value with spaces")
		return
	}
}

// TestBaggageIsSentToSubServices checks the whole point of AddToBaggage: what is put into
// the context must end up in the header of a request to a sub-service.
func TestBaggageIsSentToSubServices(t *testing.T) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	ctx, err := AddToBaggage(context.Background(), BaggageKeyInstanceId, "instance-id-1")
	if err != nil {
		t.Error(err)
		return
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/engine-rest/message", nil)
	if err != nil {
		t.Error(err)
		return
	}
	err = otelx.InjectContextToRequest(ctx, req)
	if err != nil {
		t.Error(err)
		return
	}

	header := req.Header.Get("baggage")
	if !strings.Contains(header, BaggageKeyInstanceId+"=instance-id-1") {
		t.Error(header)
		return
	}
}
