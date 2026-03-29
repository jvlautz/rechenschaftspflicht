package handlers

import (
	"log"
	"net/http"

	"github.com/erkannt/rechenschaftspflicht/views"
	"github.com/julienschmidt/httprouter"
)

func OopsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.WriteHeader(http.StatusInternalServerError)

	err := views.LayoutBare(views.Oops()).Render(r.Context(), w)

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error rendering layout: %v", err)
		return
	}
}
