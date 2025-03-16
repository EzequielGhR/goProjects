package models

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/go-chi/render"
	"github.com/google/uuid"
)

// TODO: Implement storage
const StoragePath = "./storage"

// Context key for page context
type ContextKey string

const CtxKey ContextKey = "page"

// <<< Errors >>>
type ErrResponse struct {
	Err            error
	HTTPStatusCode int
	StatusText     string
	ErrorText      string
}

func (e *ErrResponse) Render(wr http.ResponseWriter, req *http.Request) error {
	render.Status(req, e.HTTPStatusCode)
	return nil
}

// 400
func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusBadRequest,
		StatusText:     "Invalid request",
		ErrorText:      err.Error(),
	}
}

// 422
func ErrRender(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusUnprocessableEntity,
		StatusText:     "Error rendering response",
		ErrorText:      err.Error(),
	}
}

// 404
var ErrNotFound = &ErrResponse{
	HTTPStatusCode: http.StatusNotFound,
	StatusText:     "Resource not found",
}

// <<< Pages >>>
type Page struct {
	ID    string
	Title string
	Path  string
}

// TODO: Non Static Pages
var pages = []*Page{
	{ID: "1", Title: "Main page", Path: path.Join(StoragePath, "1.txt")},
	{ID: "2", Title: "Test Page", Path: path.Join(StoragePath, "2.txt")},
}

// <<< Request and Response structs >>>
// Requests
type PageRequest struct {
	*Page

	ProtectedID string
}

func (pageReq *PageRequest) Bind(req *http.Request) error {
	if pageReq.Page == nil {
		return errors.New("missing required page fields")
	}

	// TODO: Something with this protected id.
	pageReq.ProtectedID = ""
	pageReq.Page.Title = strings.ToLower(pageReq.Page.Title)
	return nil
}

// Response
type PageResponse struct {
	*Page

	Elapsed int64
}

func (pageResp *PageResponse) Render(wr http.ResponseWriter, req *http.Request) error {
	// TODO: Dynamic elapsed time
	pageResp.Elapsed = 10
	return nil
}

func NewPageResponse(page *Page) *PageResponse {
	resp := &PageResponse{Page: page}
	return resp
}

func NewPageListResponse(pages []*Page) []render.Renderer {
	pageList := []render.Renderer{}
	for _, page := range pages {
		pageList = append(pageList, NewPageResponse(page))
	}
	return pageList
}

// <<< internal functions >>>
func InternalCreateNewPage(page *Page) (string, error) {
	page.ID = uuid.New().String()
	pages = append(pages, page)
	return page.ID, nil
}

func InternalGetAllPages() ([]*Page, error) {
	return pages, nil
}

func InternalGetPage(pageID string) (*Page, error) {
	for _, page := range pages {
		if page.ID == pageID {
			return page, nil
		}
	}

	return nil, fmt.Errorf(
		"page with id '%s' not found",
		pageID,
	)
}

func InternalUpdatePage(pageID string, page *Page) (*Page, error) {
	for i, p := range pages {
		if p.ID == pageID {
			pages[i] = page
			return page, nil
		}
	}

	return nil, fmt.Errorf(
		"page with id '%s' not found",
		pageID,
	)
}

func InternalDeletePage(pageID string) (*Page, error) {
	for i, p := range pages {
		if p.ID == pageID {
			pages = slices.Delete(pages, i, i)
			return p, nil
		}
	}

	return nil, errors.New("page not found")
}
