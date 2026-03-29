package handlers

import (
	"log"
	"net/http"

	"github.com/erkannt/rechenschaftspflicht/services/authentication"
	"github.com/erkannt/rechenschaftspflicht/services/userstore"
	"github.com/erkannt/rechenschaftspflicht/views"
	"github.com/julienschmidt/httprouter"
)

func LandingHandler(auth authentication.Auth) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if auth.IsLoggedIn(r) {
			cookie, _ := r.Cookie("auth")
			email, _ := auth.ValidateToken(cookie.Value)
			log.Printf("User %s already logged in, redirecting to /record-event", email)
			http.Redirect(w, r, "/record-event", http.StatusFound)
			return
		}
		err := views.LayoutBare(views.Login()).Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error rendering layout: %v", err)
			return
		}
	}
}

func LoginPostHandler(userStore userstore.UserStore, auth authentication.Auth) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if err := r.ParseForm(); err != nil {
			log.Printf("error parsing form: %v", err)
			http.Redirect(w, r, "/oops", http.StatusFound)
			return
		}

		email := r.FormValue("email")
		if email == "" {
			log.Println("email required")
			http.Redirect(w, r, "/oops", http.StatusFound)
			return
		}

		exists, err := userStore.IsUser(email)
		if err != nil {
			log.Printf("error checking user %s: %v", email, err)
			http.Redirect(w, r, "/oops", http.StatusFound)
			return
		}
		if !exists {
			log.Printf("unauthorized email attempt: %s", email)
			http.Redirect(w, r, "/check-your-email", http.StatusFound)
			return
		}

		token, err := auth.GenerateToken(email)
		if err != nil {
			log.Printf("could not generate token for %s: %v", email, err)
			http.Redirect(w, r, "/oops", http.StatusFound)
			return
		}
		if err := auth.SendMagicLink(email, token); err != nil {
			log.Printf("could not send email to %s: %v", email, err)
			http.Redirect(w, r, "/oops", http.StatusFound)
			return
		}
		log.Printf("magic login link sent to %s", email)

		http.Redirect(w, r, "/check-your-email", http.StatusFound)
	}
}

func LoginGetHandler(auth authentication.Auth) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		email, err := auth.ValidateToken(token)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		cookie := auth.LoggedIn(token)
		http.SetCookie(w, &cookie)

		log.Printf("User %s logged in via magic link", email)
		http.Redirect(w, r, "/all-events", http.StatusFound)
	}
}

func LogoutHandler(auth authentication.Auth) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		cookie := auth.LoggedOut()
		http.SetCookie(w, &cookie)

		log.Println("User logged out")
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func CheckYourEmailHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	err := views.LayoutBare(views.CheckYourEmail()).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error rendering layout: %v", err)
		return
	}
}
