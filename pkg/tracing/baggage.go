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
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

// BaggageKeyInstanceId is the baggage key under which the smart-service-instance-id
// is passed on to sub-services and written to the logs.
const BaggageKeyInstanceId = "smart_service_instance_id"

// AddToBaggage returns a context that carries key=value in its open-telemetry baggage
// and adds the same information to the currently active span.
// gin-middleware/otelx offers this only for *http.Request; ids that are created while
// handling a request (like the smart-service-instance-id) are only known deeper in the
// call stack, where the context is the only thing that is passed on.
func AddToBaggage(ctx context.Context, key string, value string) (context.Context, error) {
	if key == "" || value == "" {
		return ctx, nil
	}
	if strings.Contains(key, " ") {
		return ctx, fmt.Errorf("baggage key contains spaces")
	}
	if strings.Contains(value, " ") {
		return ctx, fmt.Errorf("baggage value contains spaces")
	}
	member, err := baggage.NewMember(key, value)
	if err != nil {
		return ctx, fmt.Errorf("failed to create baggage member: %w", err)
	}
	bag, err := baggage.FromContext(ctx).SetMember(member)
	if err != nil {
		return ctx, fmt.Errorf("failed to set baggage member: %w", err)
	}
	ctx = baggage.ContextWithBaggage(ctx, bag)

	//keep the currently active span in sync with the baggage
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attribute.String(key, value))
	}
	return ctx, nil
}
