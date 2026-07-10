package httpdelivery

import (
	"encoding/json"
	"net/http"

	"github.com/aralary/edgeguard/internal/controlplane/usecase"
	"github.com/labstack/echo/v5"
)

func (h *Handler) createProject(c *echo.Context) error {
	var request createProjectRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&request); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body"})
	}

	project, err := h.usecase.CreateProject(c.Request().Context(), usecase.CreateProjectInput{
		Name: request.Name,
	})
	if err != nil {
		return h.handleError(c, "failed to create project", err)
	}

	return c.JSON(http.StatusCreated, newProjectResponse(project))
}

func (h *Handler) listProjects(c *echo.Context) error {
	projects, err := h.usecase.ListProjects(c.Request().Context())
	if err != nil {
		return h.handleError(c, "failed to list projects", err)
	}

	return c.JSON(http.StatusOK, newProjectListResponse(projects))
}
