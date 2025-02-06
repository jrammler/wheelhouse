package web

import (
	"net/http"

	"github.com/jrammler/wheelhouse/internal/controller/web/templates"
)

func (s *Server) AddAuthHandlers() {
	http.HandleFunc("GET /login", s.handleLoginGet)
	http.HandleFunc("POST /login", s.handleLoginPost)
	http.HandleFunc("GET /logout", s.handleLogoutGet)
}

func (s *Server) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	templates.Login(false).Render(r.Context(), w)
}

func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	username := r.Form.Get("username")
	password := r.Form.Get("password")

	sessionToken, expiration, err := s.service.AuthService.LoginUser(r.Context(), username, password)
	if err != nil {
		templates.Login(true).Render(r.Context(), w)
		return
	}

	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		HttpOnly: true,
		Path:     "/", // important to set path to root, so it is valid for all paths
		Expires:  *expiration,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleLogoutGet(w http.ResponseWriter, r *http.Request) {
	sessionCookie, err := r.Cookie("session_token")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	s.service.AuthService.LogoutUser(r.Context(), sessionCookie.Value)

	// clear session cookie
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    "",
		HttpOnly: true,
		Path:     "/",
		MaxAge:   -1, // this tells the browser to delete the cookie
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/login", http.StatusFound)
}
