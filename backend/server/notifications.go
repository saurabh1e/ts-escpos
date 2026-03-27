package server

import (
	"context"
	"fmt"
	"strings"
)

type NotificationPayload struct {
	Title   string         `json:"title"`
	Message string         `json:"message"`
	Icon    string         `json:"icon,omitempty"`
	Link    string         `json:"link,omitempty"`
	Sound   bool           `json:"sound"`
	Data    map[string]any `json:"data,omitempty"`
}

func DeliverNotification(ctx context.Context, emitter EventEmitter, payload NotificationPayload, emitFrontend bool) {
	title := strings.TrimSpace(payload.Title)
	message := strings.TrimSpace(payload.Message)
	icon := strings.TrimSpace(payload.Icon)

	if title == "" {
		title = "Notification"
	}
	if message == "" {
		message = "You have a new notification."
	}

	logMsg := fmt.Sprintf("[Notification] Title: %s | Message: %s", title, message)
	fmt.Println(logMsg)
	if ctx != nil && emitter != nil {
		emitter(ctx, "backend_log", logMsg)
	}

	if emitFrontend && ctx != nil && emitter != nil {
		emitter(ctx, "error_notification", map[string]string{
			"title":   title,
			"message": message,
			"icon":    icon,
		})
	}

	if payload.Sound {
		go playSystemSound()
	}

	go func() {
		if err := sendSystemNotification(title, message, icon); err != nil {
			fmt.Printf("Failed to send system notification: %v\n", err)
		}
	}()
}
