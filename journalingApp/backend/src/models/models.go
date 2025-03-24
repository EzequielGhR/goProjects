package models

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/render"
	"github.com/google/uuid"
)

const StoragePath = "../../storage"

const MaxByteSize = 10000

// <<< Context key for page context >>>

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

// Error 400
func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusBadRequest,
		StatusText:     "Invalid request",
		ErrorText:      err.Error(),
	}
}

// Error 422
func ErrRender(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusUnprocessableEntity,
		StatusText:     "Error rendering response",
		ErrorText:      err.Error(),
	}
}

// Error 404
var ErrNotFound = &ErrResponse{
	HTTPStatusCode: http.StatusNotFound,
	StatusText:     "Resource not found",
}

// <<< Pages >>>

type Page struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

var pages = []*Page{}

func protectedLoadPages() {
	log.Println("Loading Pages from storage")
	pagesPath := path.Join(StoragePath, "pages")
	if err := os.MkdirAll(pagesPath, os.ModePerm); err != nil {
		log.Panic(err)
	}

	files, err := os.ReadDir(pagesPath)
	if err != nil {
		log.Panic(err)
	}

	localPages := []*Page{}
	for _, file := range files {
		if file.IsDir() {
			log.Println("INFO: Skipping directory")
			continue
		}

		fileName := file.Name()
		if !strings.HasSuffix(fileName, ".txt") {
			log.Println("INFO: Skipping non txt file")
			continue
		}

		fileParts := strings.Split(fileName, ".")
		if len(fileParts) != 3 {
			log.Printf("ERROR: Malformed file name '%s'\n", fileName)
			continue
		}

		log.Printf("Found Page text file: %s\n", fileName)

		page := new(Page)
		page.Title = fileParts[0]
		page.ID = fileParts[1]
		page.Path = path.Join(pagesPath, fileName)
		localPages = append(localPages, page)
	}

	pages = localPages
	log.Println("Loaded all available pages")
}

func LoadPages(forceReload bool) []*Page {
	if len(pages) == 0 || forceReload {
		protectedLoadPages()
		return pages
	}

	return pages
}

// <<< Request and Response structs >>>

// Requests
type PageRequest struct {
	*Page

	Content string `json:"content"`
}

func (pageReq *PageRequest) Bind(req *http.Request) error {
	if pageReq.Page == nil {
		return errors.New("missing required page fields")
	}

	pageReq.Page.Title = strings.ToLower(pageReq.Page.Title)
	return nil
}

// Response
type PageResponse struct {
	*Page

	Elapsed int64
}

type PageWithContentResponse struct {
	*Page

	Content string `json:"content"`
}

func (pageResp *PageResponse) Render(wr http.ResponseWriter, req *http.Request) error {
	// TODO: Dynamic elapsed time
	pageResp.Elapsed = 10
	return nil
}

func (pageResp *PageWithContentResponse) Render(wr http.ResponseWriter, req *http.Request) error {
	// TODO: Dynamic elapsed time
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

func NewPageWithContentResponse(page *Page, content string) *PageWithContentResponse {
	resp := &PageWithContentResponse{
		Page:    page,
		Content: content,
	}
	return resp
}

// <<< internal functions >>>

func InternalCreateNewPage(page *Page, content string) (string, error) {

	page.ID = hex.EncodeToString([]byte(uuid.NewString()))
	if err := createFSPage(page, content); err != nil {
		return "", err
	}

	LoadPages(true)
	return page.ID, nil
}

func InternalGetAllPages() ([]*Page, error) {
	availablePages := LoadPages(false)
	return availablePages, nil
}

func InternalGetPage(pageID string) (*Page, error) {
	availablePages := LoadPages(false)
	for _, page := range availablePages {
		if page.ID == pageID {
			return page, nil
		}
	}

	return nil, fmt.Errorf(
		"page with id '%s' not found",
		pageID,
	)
}

func InternalUpdatePage(pageID string, page *Page, content string) (*Page, error) {
	availablePages := LoadPages(false)
	for _, p := range availablePages {
		if p.ID == pageID {
			if err := createFSPage(page, content); err != nil {
				return nil, err
			}

			LoadPages(true)
			return page, nil
		}
	}

	return nil, fmt.Errorf(
		"page with id '%s' not found",
		pageID,
	)
}

func InternalDeletePage(pageID string) (*Page, error) {
	availablePages := LoadPages(false)
	for _, page := range availablePages {
		if page.ID == pageID {
			if err := deleteFSPage(page); err != nil {
				return nil, err
			}

			LoadPages(true)
			return page, nil
		}
	}

	return nil, errors.New("page not found")
}

func InternalGetPageWithContents(pageID string) (*Page, string, error) {
	availablePages := LoadPages(false)
	for _, page := range availablePages {
		if page.ID == pageID {
			pageContent, err := readFSPage(page)
			if err != nil {
				return nil, "", err
			}

			return page, pageContent, nil
		}
	}

	return nil, "", errors.New("Page Not Found")
}

// <<< FileSystem >>>

func createFile(filePath string, content string) error {
	log.Printf("INFO: Creating file at '%s'\n", filePath)

	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("ERROR: Failed to create file '%s'\n", filePath)
		return err
	}

	defer file.Close()

	log.Println("INFO: Writing contents to file")

	_, err = file.Write([]byte(content))
	if err != nil {
		log.Println("ERROR: Failed to write contents to file")
		return err
	}

	return nil
}

func deleteFile(filePath string) error {
	log.Printf("INFO: Deleting file at '%s'\n", filePath)

	if err := os.Remove(filePath); err != nil {
		log.Printf("ERROR: Failed to delete file '%s'\n", filePath)
		return err
	}

	return nil
}

func readFile(filePath string) (string, error) {
	log.Printf("INFO: Reading file at '%s\n", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("There was an error openning file '%s'\n", filePath)
		return "", err
	}

	defer file.Close()

	buffer := make([]byte, MaxByteSize)

	bytesRead, err := bufio.NewReader(file).Read(buffer)
	if err != nil {
		return string(buffer[:bytesRead]), err
	}

	return string(buffer[:bytesRead]), nil
}

func createFSPage(page *Page, contents string) error {
	fileId := getSafeId(page)
	page.Path = path.Join(StoragePath, "pages", fileId)
	if err := createFile(page.Path, contents); err != nil {
		return err
	}

	return nil
}

func deleteFSPage(page *Page) error {
	if err := deleteFile(page.Path); err != nil {
		return err
	}

	return nil
}

func readFSPage(page *Page) (string, error) {
	content, err := readFile(page.Path)

	if err != nil {
		return content, err
	}

	return content, nil
}

func getSafeId(page *Page) string {
	// TODO: Improve this
	fileId := strings.Join([]string{page.Title, page.ID, "txt"}, ".")
	fileId = strings.ToLower(fileId)
	fileId = strings.Replace(fileId, " ", "_", -1)
	return fileId
}
