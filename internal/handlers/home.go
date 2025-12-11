package handlers

import (
	"net/http"

	"github.com/tmunongo/nanotools/web/templates"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.HomePage().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}
