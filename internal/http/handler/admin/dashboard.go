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

// UsersPage serves the users management page.
type UsersPage struct {
	templates *template.Template
}

// NewUsersPage creates a users page handler.
func NewUsersPage(templates *template.Template) *UsersPage {
	return &UsersPage{templates: templates}
}

// Index renders the users management page.
func (h *UsersPage) Index(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "admin_users.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ConnectorsPage serves the connectors management page.
type ConnectorsPage struct {
	templates *template.Template
}

// NewConnectorsPage creates a connectors page handler.
func NewConnectorsPage(templates *template.Template) *ConnectorsPage {
	return &ConnectorsPage{templates: templates}
}

// Index renders the connectors management page.
func (h *ConnectorsPage) Index(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "admin_connectors.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
