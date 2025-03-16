package routes

import (
	"context"
	"errors"
	"models"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// Main routes function to be used on main router
func PageGlobalRoutes(pageRouter chi.Router) {
	pageRouter.Get("/", ListPages)
	pageRouter.Post("/", CreatePage)
	pageRouter.Route("/{pageID}", pageElementRoutes)
}

func pageElementRoutes(elementRouter chi.Router) {
	// Use the page context middleware
	elementRouter.Use(PageCtx)

	elementRouter.Get("/", GetPage)
	elementRouter.Put("/", UpdatePage)
	elementRouter.Delete("/", DeletePage)
}

// <<< Endpoints >>>

func ListPages(wr http.ResponseWriter, req *http.Request) {
	pages, err := models.InternalGetAllPages()
	if err != nil {
		render.Render(wr, req, models.ErrRender(err))
		return
	}

	if err = render.RenderList(wr, req, models.NewPageListResponse(pages)); err != nil {
		render.Render(wr, req, models.ErrRender(err))
		return
	}
}

func CreatePage(wr http.ResponseWriter, req *http.Request) {
	data := new(models.PageRequest)
	if err := render.Bind(req, data); err != nil {
		render.Render(wr, req, models.ErrInvalidRequest(err))
		return
	}

	page := data.Page
	_, err := models.InternalCreateNewPage(page)
	if err != nil {
		render.Render(wr, req, models.ErrRender(err))
		return
	}

	render.Status(req, http.StatusCreated)
	render.Render(wr, req, models.NewPageResponse(page))
}

// The page context middleware
func PageCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(wr http.ResponseWriter, req *http.Request) {
			// Get the pageID from the URL params
			pageID := chi.URLParam(req, "pageID")
			if pageID == "" {
				render.Render(wr, req, models.ErrNotFound)
				return
			}

			page, err := models.InternalGetPage(pageID)
			if err != nil {
				render.Render(wr, req, models.ErrNotFound)
				return
			}

			// Use a child context with the page as value
			ctx := context.WithValue(req.Context(), models.CtxKey, page)
			next.ServeHTTP(wr, req.WithContext(ctx))
		},
	)
}

func GetPage(wr http.ResponseWriter, req *http.Request) {
	// Get the page from the context, this is possible because of the PageCtx middleware
	page, ok := req.Context().Value(models.CtxKey).(*models.Page)
	if !ok {
		render.Render(
			wr,
			req,
			models.ErrRender(
				errors.New("context does not hold a Page element"),
			),
		)
	}

	if err := render.Render(wr, req, models.NewPageResponse(page)); err != nil {
		render.Render(wr, req, models.ErrRender(err))
		return
	}
}

func UpdatePage(wr http.ResponseWriter, req *http.Request) {
	// Get the page from the context, this is possible because of the PageCtx middleware
	page, ok := req.Context().Value(models.CtxKey).(*models.Page)
	if !ok {
		render.Render(
			wr,
			req,
			models.ErrRender(
				errors.New("context does not hold a Page element"),
			),
		)
	}

	data := &models.PageRequest{Page: page}
	if err := render.Bind(req, data); err != nil {
		render.Render(wr, req, models.ErrInvalidRequest(err))
		return
	}

	_, err := models.InternalUpdatePage(page.ID, page)
	if err != nil {
		render.Render(wr, req, models.ErrRender(err))
		return
	}

	render.Render(wr, req, models.NewPageResponse(page))
}

func DeletePage(wr http.ResponseWriter, req *http.Request) {
	// Get the page from the context, this is possible because of the PageCtx middleware
	page, ok := req.Context().Value(models.CtxKey).(*models.Page)
	if !ok {
		render.Render(
			wr,
			req,
			models.ErrRender(
				errors.New("context does not hold a Page element"),
			),
		)
	}

	page, err := models.InternalDeletePage(page.ID)
	if err != nil {
		render.Render(wr, req, models.ErrInvalidRequest(err))
		return
	}

	render.Render(wr, req, models.NewPageResponse(page))
}
