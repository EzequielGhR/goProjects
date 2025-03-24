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

// Track the pages globally
var pages = []*Page{}

/*
Load pages from local storage files. Files with format
page_title.page_id.txt are loaded
*/
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

/*
Load pages array, being cached or force reloaded

	forceReload: Force global var with pages to be reloaded
	return: A slice of pages available
*/
func LoadPages(forceReload bool) []*Page {
	if len(pages) == 0 || forceReload {
		protectedLoadPages()
		return pages
	}

	return pages
}

// <<< Request and Response structs >>>

// Requests and response
type PageWithContent struct {
	*Page

	Content string `json:"content"`
}

func (pageReq *PageWithContent) Bind(req *http.Request) error {
	if pageReq.Page == nil {
		return errors.New("missing required page fields")
	}

	pageReq.Page.Title = strings.ToLower(pageReq.Page.Title)
	return nil
}

func (pageResp *PageWithContent) Render(wr http.ResponseWriter, req *http.Request) error {
	// TODO: Dynamic elapsed time
	return nil
}

// Response without content
type PageResponse struct {
	*Page

	Elapsed int64
}

func (pageResp *PageResponse) Render(wr http.ResponseWriter, req *http.Request) error {
	// TODO: Dynamic elapsed time
	pageResp.Elapsed = 10
	return nil
}

/*
Return a PageResponse struct reference to be rendered

	page: The pointer to the Page object to be rendered
*/
func NewPageResponse(page *Page) *PageResponse {
	resp := &PageResponse{Page: page}
	return resp
}

/*
Return an array of renderers for listing pages

	pages: The slice of pointers to pages to be rendered
*/
func NewPageListResponse(pages []*Page) []render.Renderer {
	pageList := []render.Renderer{}
	for _, page := range pages {
		pageList = append(pageList, NewPageResponse(page))
	}
	return pageList
}

/*
Return a pointer to a page with content struct to be rendered.

	page: The reference to a page to be rendered.
	content: The content of the page
*/
func NewPageWithContentResponse(page *Page, content string) *PageWithContent {
	resp := &PageWithContent{
		Page:    page,
		Content: content,
	}
	return resp
}

// <<< internal functions >>>

/*
Create a new page on storage

	page: The reference to the page to be stored.
	content: The content of the page to store.
	return: The page ID or an empty string, with an error or nil on success
*/
func InternalCreateNewPage(page *Page, content string) (string, error) {
	page.ID = hex.EncodeToString([]byte(uuid.NewString()))
	if err := createFSPage(page, content); err != nil {
		return "", err
	}

	LoadPages(true)
	return page.ID, nil
}

/*
Fetch a page from storage.

	pageID: The id of the page to fetch.
	return: A reference to the page fetched, plus an error which is nil on success.
*/
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

/*
Update a page stored by ID.

	pageID: The id of the page to update.
	page: The reference to the page to be created.
	content: The new content for the updated file
	return: A reference to the updated page and an error which is nil on success.
*/
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

/*
Delete a page from storage

	pageID: The id of the page to delete.
	return: A reference to the deleted page, plus an error which is nil on success.
*/
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

/*
Get a page and its contents from storage.

	pageID: The id of the page to fetch.
	return: A reference to the page, its content and an error which is nil on success.
*/
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

/*
Create a file on the system

	filePath:
	content:
	return: An error, nil on success
*/
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

/*
Delete a file on the system

	filePath:
	return: An error, nil on success.
*/
func deleteFile(filePath string) error {
	log.Printf("INFO: Deleting file at '%s'\n", filePath)

	if err := os.Remove(filePath); err != nil {
		log.Printf("ERROR: Failed to delete file '%s'\n", filePath)
		return err
	}

	return nil
}

/*
Read a file from the system

	filePath:
	return: The file contents and an error, which is nil on success
*/
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

/*
Create a page with the correct format, on the file system.

	page: A reference to te page to save.
	contents: The contents of the page.
	return: An error which is nil on success.
*/
func createFSPage(page *Page, contents string) error {
	fileId := getSafeId(page)
	page.Path = path.Join(StoragePath, "pages", fileId)
	if err := createFile(page.Path, contents); err != nil {
		return err
	}

	return nil
}

/*
Delete a page from the file system

	page: A reference to the page to be deleted
	return: An error, nil on success.
*/
func deleteFSPage(page *Page) error {
	if err := deleteFile(page.Path); err != nil {
		return err
	}

	return nil
}

/*
Read a page contents from the file system.

	page: A reference to the page to read.
	return: The contents of the page and an error, which is nil on success.
*/
func readFSPage(page *Page) (string, error) {
	content, err := readFile(page.Path)

	if err != nil {
		return content, err
	}

	return content, nil
}

/*
Sanitize the file id of a page based on its title and page ID.

	page: A reference to the page to sanitize the file id of.
	return: The sanitized id
*/
func getSafeId(page *Page) string {
	// TODO: Improve this
	fileId := strings.Join([]string{page.Title, page.ID, "txt"}, ".")
	fileId = strings.ToLower(fileId)
	fileId = strings.Replace(fileId, " ", "_", -1)
	return fileId
}
