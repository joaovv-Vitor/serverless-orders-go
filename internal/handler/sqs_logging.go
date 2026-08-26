package handler

import "github.com/joaovv-Vitor/serverless-orders-go/internal/domain"

func sqsLogAttributes(service, messageID string, event domain.OrderCreatedEvent) []any {
	attributes := []any{
		"service", service,
		"messageId", messageID,
	}
	if event.EventID != "" {
		attributes = append(attributes,
			"eventId", event.EventID,
			"orderId", event.Data.OrderID,
			"eventType", event.EventType,
		)
	}
	return attributes
}
