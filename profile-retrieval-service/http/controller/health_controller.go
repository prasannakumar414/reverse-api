package controller

import (
	"net/http"

	"github.com/prasannakumar414/profile-retrieval-service/http/helper"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

func (c *HealthController) Health(w http.ResponseWriter, _ *http.Request) {
	helper.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
