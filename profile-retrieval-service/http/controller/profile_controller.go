package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/prasannakumar414/profile-retrieval-service/http/helper"
	"github.com/prasannakumar414/profile-retrieval-service/services"
)

type ProfileController struct {
	service *services.ProfileService
}

type retrieveProfileRequest struct {
	ProfileURL string `json:"profile_url"`
	URL        string `json:"url"`
}

func NewProfileController(service *services.ProfileService) *ProfileController {
	return &ProfileController{
		service: service,
	}
}

func (c *ProfileController) Retrieve(w http.ResponseWriter, r *http.Request) {
	profileURL, ok := c.profileURLFromRequest(w, r)
	if !ok {
		return
	}

	profile, err := c.service.Retrieve(r.Context(), profileURL)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidProfileURL):
			helper.WriteError(w, http.StatusBadRequest, "profile_url must be a linkedin /in/ profile URL")
		case errors.Is(err, services.ErrFetchFailed):
			helper.WriteError(w, http.StatusBadGateway, err.Error())
		default:
			helper.WriteError(w, http.StatusInternalServerError, "failed to retrieve profile")
		}
		return
	}

	helper.WriteJSON(w, http.StatusOK, profile)
}

func (c *ProfileController) profileURLFromRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	switch r.Method {
	case http.MethodGet:
		profileURL := r.URL.Query().Get("profile_url")
		if profileURL == "" {
			profileURL = r.URL.Query().Get("url")
		}
		if profileURL == "" {
			helper.WriteError(w, http.StatusBadRequest, "profile_url query parameter is required")
			return "", false
		}
		return profileURL, true

	case http.MethodPost:
		defer r.Body.Close()

		var request retrieveProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			helper.WriteError(w, http.StatusBadRequest, "invalid json body")
			return "", false
		}

		profileURL := request.ProfileURL
		if profileURL == "" {
			profileURL = request.URL
		}
		if profileURL == "" {
			helper.WriteError(w, http.StatusBadRequest, "profile_url is required")
			return "", false
		}
		return profileURL, true

	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		helper.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return "", false
	}
}
