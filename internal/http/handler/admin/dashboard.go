package admin

import (
	"html/template"
	"net/http"
)

// Dashboard serves the admin dashboard page.
type Dashboard struct {
	templates *template.Template
}

// NewDashboard creates an admin dashboard handler.
func NewDashboard(templates *template.Template) *Dashboard {
	return &Dashboard{templates: templates}
}

// Index renders the admin dashboard home page.
func (h *Dashboard) Index(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"UserCount":      0, // TODO: wire actual repo calls
		"ConnectorCount": 0,
		"AIConfigCount":  0,
	}
	if err := h.templates.ExecuteTemplate(w, "admin_dashboard.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
