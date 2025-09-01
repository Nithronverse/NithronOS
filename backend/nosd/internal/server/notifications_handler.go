package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"nithronos/backend/nosd/internal/notifications"
	"nithronos/backend/nosd/pkg/httpx"
)

// NotificationHandler handles notification API endpoints
type NotificationHandler struct {
	manager *notifications.Manager
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(manager *notifications.Manager) *NotificationHandler {
	return &NotificationHandler{
		manager: manager,
	}
}

// Routes registers notification routes
func (h *NotificationHandler) Routes() chi.Router {
	r := chi.NewRouter()
	
	// Notifications
	r.Get("/", h.ListNotifications)
	r.Get("/{id}", h.GetNotification)
	r.Put("/{id}/read", h.MarkRead)
	r.Put("/read-all", h.MarkAllRead)
	r.Delete("/{id}", h.DeleteNotification)
	r.Get("/subscribe", h.Subscribe)
	
	// Channels
	r.Get("/channels", h.ListChannels)
	r.Post("/channels", h.CreateChannel)
	r.Get("/channels/{id}", h.GetChannel)
	r.Put("/channels/{id}", h.UpdateChannel)
	r.Delete("/channels/{id}", h.DeleteChannel)
	r.Post("/channels/{id}/test", h.TestChannel)
	
	return r
}

// ListNotifications returns all notifications
func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	unreadOnly := r.URL.Query().Get("unread") == "true"
	notifications := h.manager.List(unreadOnly)
	httpx.WriteJSON(w, notifications, http.StatusOK)
}

// GetNotification returns a specific notification
func (h *NotificationHandler) GetNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	notif, ok := h.manager.Get(id)
	if !ok {
		httpx.WriteError(w, "NOT_FOUND", "Notification not found", http.StatusNotFound)
		return
	}
	
	httpx.WriteJSON(w, notif, http.StatusOK)
}

// MarkRead marks a notification as read
func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	if err := h.manager.MarkRead(id); err != nil {
		httpx.WriteError(w, "NOT_FOUND", "Notification not found", http.StatusNotFound)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// MarkAllRead marks all notifications as read
func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	if err := h.manager.MarkAllRead(); err != nil {
		httpx.WriteError(w, "UPDATE_FAILED", "Failed to mark notifications as read", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// DeleteNotification deletes a notification
func (h *NotificationHandler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	if err := h.manager.Delete(id); err != nil {
		httpx.WriteError(w, "DELETE_FAILED", "Failed to delete notification", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// Subscribe creates a real-time subscription for notifications
func (h *NotificationHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	
	// Get client ID from session
	clientID := r.Header.Get("X-UID")
	if clientID == "" {
		clientID = "anonymous"
	}
	
	// Create subscription
	ch := h.manager.Subscribe(clientID)
	defer h.manager.Unsubscribe(clientID, ch)
	
	// Send initial ping
	w.Write([]byte("event: ping\ndata: {}\n\n"))
	w.(http.Flusher).Flush()
	
	// Listen for notifications
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case notif := <-ch:
			data, _ := json.Marshal(notif)
			w.Write([]byte("event: notification\ndata: "))
			w.Write(data)
			w.Write([]byte("\n\n"))
			w.(http.Flusher).Flush()
			
		case <-ticker.C:
			// Send keepalive
			w.Write([]byte("event: ping\ndata: {}\n\n"))
			w.(http.Flusher).Flush()
			
		case <-r.Context().Done():
			return
		}
	}
}

// ListChannels returns all notification channels
func (h *NotificationHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
	channels := h.manager.ListChannels()
	
	// Hide sensitive config
	for _, ch := range channels {
		ch.Config = h.sanitizeConfig(ch.Config, ch.Type)
	}
	
	httpx.WriteJSON(w, channels, http.StatusOK)
}

// GetChannel returns a specific channel
func (h *NotificationHandler) GetChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	channel, ok := h.manager.GetChannel(id)
	if !ok {
		httpx.WriteError(w, "NOT_FOUND", "Channel not found", http.StatusNotFound)
		return
	}
	
	// Hide sensitive config
	channel.Config = h.sanitizeConfig(channel.Config, channel.Type)
	httpx.WriteJSON(w, channel, http.StatusOK)
}

// CreateChannel creates a new notification channel
func (h *NotificationHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	var channel notifications.Channel
	if err := json.NewDecoder(r.Body).Decode(&channel); err != nil {
		httpx.WriteError(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if err := h.manager.CreateChannel(&channel); err != nil {
		httpx.WriteError(w, "CREATE_FAILED", "Failed to create channel", http.StatusInternalServerError)
		return
	}
	
	// Hide sensitive config
	channel.Config = h.sanitizeConfig(channel.Config, channel.Type)
	httpx.WriteJSON(w, channel, http.StatusCreated)
}

// UpdateChannel updates a notification channel
func (h *NotificationHandler) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	var updates notifications.Channel
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		httpx.WriteError(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if err := h.manager.UpdateChannel(id, &updates); err != nil {
		httpx.WriteError(w, "UPDATE_FAILED", "Failed to update channel", http.StatusInternalServerError)
		return
	}
	
	channel, _ := h.manager.GetChannel(id)
	channel.Config = h.sanitizeConfig(channel.Config, channel.Type)
	httpx.WriteJSON(w, channel, http.StatusOK)
}

// DeleteChannel deletes a notification channel
func (h *NotificationHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	if err := h.manager.DeleteChannel(id); err != nil {
		httpx.WriteError(w, "DELETE_FAILED", "Failed to delete channel", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// TestChannel tests a notification channel
func (h *NotificationHandler) TestChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	if err := h.manager.TestChannel(id); err != nil {
		httpx.WriteError(w, "TEST_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}
	
	httpx.WriteJSON(w, map[string]interface{}{
		"success": true,
		"message": "Test notification sent successfully",
	}, http.StatusOK)
}

// sanitizeConfig removes sensitive information from config
func (h *NotificationHandler) sanitizeConfig(config map[string]interface{}, channelType string) map[string]interface{} {
	sanitized := make(map[string]interface{})
	for k, v := range config {
		switch k {
		case "password", "apiKey", "token", "secret":
			sanitized[k] = "***"
		default:
			sanitized[k] = v
		}
	}
	return sanitized
}
