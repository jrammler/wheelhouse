package web

import (
	"context"
	"errors"
	"net/http"

	"github.com/jrammler/wheelhouse/internal/controller/web/templates"
	"github.com/jrammler/wheelhouse/internal/entity"
	"github.com/jrammler/wheelhouse/internal/service"
)

type userContextKeyType int

const userContextKey userContextKeyType = 0

func addUser(r *http.Request, user entity.User) *http.Request {
	c := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(c)
}

func GetUser(ctx context.Context) (entity.User, error) {
	u := ctx.Value(userContextKey)
	if u == nil {
		return entity.User{}, errors.New("Request context does not contain user")
	}
	return u.(entity.User), nil
}

func SetupAuthentication(service *service.Service, mux *http.ServeMux) *http.ServeMux {
	mux.HandleFunc("GET /login", handleLoginGet)
	mux.HandleFunc("POST /login", handleLoginPost(service))
	mux.HandleFunc("GET /logout", handleLogoutGet(service))
	authenticatedMux := http.NewServeMux()
	mux.HandleFunc("/", authenticationMiddleware(service, authenticatedMux))
	return authenticatedMux
}

func handleLoginGet(w http.ResponseWriter, r *http.Request) {
	templates.Login(false).Render(r.Context(), w)
}

func handleLoginPost(service *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}
		username := r.Form.Get("username")
		password := r.Form.Get("password")

		sessionToken, expiration, err := service.AuthService.LoginUser(r.Context(), username, password)
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
}

func handleLogoutGet(service *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		service.AuthService.LogoutUser(r.Context(), sessionCookie.Value)

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
}

func authenticationMiddleware(service *service.Service, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_token")
		if err != nil {
			w.Header().Add("Location", "/login")
			w.WriteHeader(http.StatusFound)
			return
		}
		sessionToken := sessionCookie.Value
		user, err := service.AuthService.GetSessionUser(r.Context(), sessionToken)
		if err != nil {
			w.Header().Add("Location", "/login")
			w.WriteHeader(http.StatusFound)
			return
		}
		next.ServeHTTP(w, addUser(r, user))
	}
}
