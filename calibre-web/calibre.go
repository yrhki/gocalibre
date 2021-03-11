package calibre

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/yrhki/gocalibre/calibre-web/uploadcontent"
	"github.com/PuerkitoBio/goquery"
)

type uploadResponse struct {
	Location string `json:"location"`
}

type author struct {
	Name string `json:"name"`
}

type authorReponse struct {
	Authors []author
}

type API struct {
	url string
	c *http.Client
}

func (api *API) Login(username, password string) error {
	data := url.Values{
		"next": {"/me"},
		"username": {username},
		"password": {password},
		"remember_me": {"on"},
	}
	resp, err := api.c.PostForm(api.url + "/login", data)
	if err != nil { return err }
	defer resp.Body.Close()
	return checkFlashAlert(resp)
}

func (api *API) Logout() error {
	resp, err := api.c.Get(api.url + "/logout")
	if err != nil { return err }
	defer resp.Body.Close()
	return nil
}

func printBody(body io.Reader) error {
	b, err := ioutil.ReadAll(body)
	if err != nil { return err }
	fmt.Println(string(b))
	return nil
}

func (api *API) Admin() error {
	resp, err := api.c.Get(api.url + "/admin/view")
	if err != nil { return err }
	defer resp.Body.Close()
	return nil
}

// Use API.GetCategories instead
//
// func (api *API) Categories() ([]*Category, error) {
// 	resp, err := api.c.Get(api.url + "/category")
// 	if err != nil { return nil, err }
// 	defer resp.Body.Close()
// 
// 	doc, err := goquery.NewDocumentFromReader(resp.Body)
// 	if err != nil { return nil, err }
// 
// 	categories := []*Category{}
// 
// 	doc.Find(".container .row").Each(func(i int, s *goquery.Selection) {
// 		count, err := strconv.Atoi(s.Find(".badge").Text())
// 		if err != nil { panic(err) }
// 
// 		link := s.Find("a")
// 		category := strings.TrimSpace(link.Text())
// 
// 		ids, exists := link.Attr("href")
// 		if !exists { panic("could not parse category id") }
// 		id, err := strconv.Atoi(filepath.Base(ids))
// 		if err != nil { panic(err) }
// 
// 		categories = append(categories, &Category{id, count, category})
// 	})
// 
// 	return categories, nil
// }

func (api *API) getAPI(url string) ([]string, error) {
	resp, err := api.c.Get(api.url + url)
	if err != nil { return nil, nil }
	defer resp.Body.Close()
	var authors []map[string]string
	err = json.NewDecoder(resp.Body).Decode(&authors)
	if err != nil { return nil, nil }

	var result []string
	for _, author := range authors {
		result = append(result, author["name"])
	}
	return result, nil
}

func (api *API) GetCategories() ([]string, error) { return api.getAPI("/get_tags_json") }
func (api *API) GetAuthors() ([]string, error) { return api.getAPI("/get_authors_json") }
func (api *API) GetLanguages() ([]string, error) { return api.getAPI("/get_languages_json") }
func (api *API) GetSeries() ([]string, error) { return api.getAPI("/get_series_json") }

func (api *API) ListBooks() ([]*ListBook, error) {

	books := []*ListBook{}

	for i := 1; ; i++ {
		resp, err := api.c.Get(fmt.Sprintf("%s/root/old/1/%d", api.url, i))
		if err != nil { return nil, err }
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil { return nil, err }


		doc.Find(".book").Each(func(_ int, s *goquery.Selection) {
			title := s.Find(".title").Text()
			ids, hasid := s.Find(".meta a").Attr("href")
			if !hasid { panic("unable to parse book id") }
			bookID, err := strconv.ParseUint(filepath.Base(ids), 10, 0)
			if err != nil { panic(err) }

			authors := []Author{}

			s.Find(".author-name").Each(func(_ int, s *goquery.Selection) {
				item, err := parseListItem(s)
				if err != nil { panic(err) }
				authors = append(authors, Author{id:item.ID(), name:item.Name()})
			})

			books = append(books, &ListBook{id:bookID, name:title, authors:authors})
		})

		if doc.Find(".next").Text() == "" { break }
	}
	return books, nil
}

func parseListItem(s *goquery.Selection) (ListItem, error) {
	id, hasid := s.Attr("href")
	if !hasid { panic("unable to parse authorID id") }
	authorID, err := strconv.ParseUint(filepath.Base(id), 10, 0)
	if err != nil { panic(err) }
	return &Author{id:authorID, name:s.Text()}, nil
}

func (api *API) BookByID(id uint64) (*Book, error) {
	resp, err := api.c.Get(fmt.Sprintf("%s/book/%d", api.url, id))
	if err != nil { return nil, err }
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil { return nil, err }

	title := doc.Find("h2#title").Text()
	authors, categories:= []string{}, []string{}

	// Authors
	doc.Find(".author a").Each(func(_ int, s *goquery.Selection) {
		item, err := parseListItem(s)
		if err != nil { panic(err) }
		authors = append(authors, item.Name())
	})

	// Categories
	doc.Find(".tags a").Each(func(_ int, s *goquery.Selection) {
		item, err := parseListItem(s)
		if err != nil { panic(err) }
		categories = append(categories, item.Name())
	})

	// Publisher
	publisher := doc.Find(".publishers a").Text()

	// Description
	c := doc.Find(".comments")
	c.Children().First().Remove()
	description, err := c.Html()
	if err != nil { panic(err) }
	description = strings.TrimSpace(description)

	book := &Book{
		id:id,
		Title:title,
		Description:description,
		Authors:authors,
		Categories:categories,
		Publisher:publisher,
	}
	book.Identifiers = make(BookIdentifiers)
	book.formats = make(map[Format]bool)

	// Published
	if published := doc.Find(".publishing-date p"); len(published.Nodes) > 0 {
		t, err := time.Parse("Jan _2, 2006 ", published.Text()[11:])
		if err != nil { return nil, err }
		book.Published = &t
	}

	// Rating
	book.Rating = uint8(doc.Find(".rating .good").Length())

	// Formats
	doc.Find(".btn-group").Children().First().Find("a").Each(func(_ int, s *goquery.Selection) {
		sp := strings.Split(strings.TrimSpace(s.Text()), " ")
		switch sp[0] {
		case "PDF":
			book.formats[FormatPDF] = true
		case "MOBI":
			book.formats[FormatPDF] = true
		case "EPUB":
			book.formats[FormatEPUB] = true
		case "CBZ":
			book.formats[FormatCBZ] = true
		default:
			panic("unhandled book format " + sp[0])
		}
	})

	// Series and Series Index
	// TODO:Could be unstable
	s := doc.Find("h2#title").SiblingsFiltered("p").Last()
	if class, _ := s.Last().Attr("class"); class != "author" {
		sp := strings.Split(s.Text(), " ")
		series := strings.Join(sp[3:], " ")
		seriesID, err := strconv.ParseFloat(sp[1], 0)
		if err != nil { panic("failed to parse series id") }

		book.Series = series
		book.SeriesIndex = seriesID
	}

	// Languages
	if lang := doc.Find(".languages span"); len(lang.Nodes) > 0 {
		book.Languages = strings.Split(lang.Text()[10:], ", ")
	}

	doc.Find(".identifiers a").Each(func(_ int, s *goquery.Selection) {
		v, hasLink := s.Attr("href")
		t := strings.ToLower(s.Text())
		if hasLink {
			// Type needs special convertions
			if strings.HasPrefix(t, "amazon.") {
				t = "amazon_" + t[7:]
			} else if t == "литрес" {
				t = "litres"
			}
			//book.Identifiers = append(book.Identifiers, BookIdentifier{Type:t, Value:link})
			book.Identifiers[t] = v
		}
	})

	return book, nil
}

func (api *API) loadURI(uri string) (uploadcontent.Content, error) {
	if strings.HasPrefix(uri, "http") {
		resp, err := api.c.Get(uri)
		if err != nil { return nil, err }
		return uploadcontent.ContentFromResponse(resp, true), nil
	} else {
		file, err := os.Open(uri)
		if err != nil { return nil, err }
		return uploadcontent.ContentFromFile(file, true)
	}
}

func (api *API) Upload(uri string) (*Book, error) {
	content, err := api.loadURI(uri)
	if err != nil { return nil, err }
	defer content.Close()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("btn-upload", content.Filename())
	if err != nil { return nil, err }

	_, err = io.Copy(fw, content.Reader())
	if err != nil { return nil, err }

	err = w.Close()
	if err != nil { return nil, err }

	resp, err := api.c.Post(api.url + "/upload", w.FormDataContentType(), &b)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	//var p bytes.Buffer
	//_, err = io.Copy(&p, resp.Body)
	//if err != nil { return nil, err }

	uploadResp := new(uploadResponse)
	err = json.NewDecoder(resp.Body).Decode(uploadResp)
	if err != nil { return nil, err }

	id, err := strconv.ParseUint(filepath.Base(uploadResp.Location), 10, 0)
	if err != nil { return nil, err }

	return api.BookByID(id)
}

func (api *API) UpdateBookCover(id uint64, uri string) error {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if strings.HasPrefix(uri, "http") {
		err := w.WriteField("cover_url", uri)
		if err != nil { return err }
	} else {
		file, err := os.Open(uri)
		if err != nil { return err }
		defer file.Close()

		// Detect ContentType
		b := make([]byte, 512)
		_, err = file.Read(b)
		if err != nil { return err }
		contentType := http.DetectContentType(b)

		_, err = file.Seek(0,0)
		if err != nil { return err }

		// Check if file has valid ContentType
		if !(contentType == "image/png" || contentType == "image/jpeg" || contentType == "image/webp") {
			return errors.New("Invalid Content-Type")
		}

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="btn-upload-cover"; filename="%s"`, filepath.Base(uri)))
		h.Set("Content-Type", contentType)
		fw, err := w.CreatePart(h)
		if err != nil { return err }

		_, err = io.Copy(fw, file)
		if err != nil { return err }
	}

	err := w.Close()
	if err != nil { return err }


	resp, err := api.c.Post(fmt.Sprintf("%s/admin/book/%d", api.url, id), w.FormDataContentType(), &b)
	if err != nil { return err }
	defer resp.Body.Close()
	return nil
}



func (api *API) UpdateBookMetadata(book *Book) error {
	w, b, err := book.multipart()
	if err != nil { return err }

	err = w.Close()
	if err != nil { return err }

	resp, err := api.c.Post(fmt.Sprintf("%s/admin/book/%d", api.url, book.id), w.FormDataContentType(), b)
	if err != nil { return err }
	defer resp.Body.Close()
	err = checkFlashAlert(resp)
	if err != nil { return err }
	return nil
}

func (api *API) BookUploadFormat(book *Book, uri string) error {
	r, err := api.loadURI(uri)
	if err != nil { return err }
	defer r.Close()

	w, b, err := book.multipart()

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="btn-upload-format"; filename="%s"`, r.Filename()))
	h.Set("Content-Type", r.ContentType())
	f, err := w.CreatePart(h)
	if err != nil { return err }
	_, err = io.Copy(f, r.Reader())
	if err != nil { return err }

	err = w.Close()
	if err != nil { return err }


	resp, err := api.c.Post(fmt.Sprintf("%s/admin/book/%d", api.url, book.id), w.FormDataContentType(), b)
	if err != nil { return err }
	defer resp.Body.Close()
	err = checkFlashAlert(resp)
	if err != nil { return err }
	return nil
}

func (api *API) UploadFormat(id uint64, uri string) error {
	book, err := api.BookByID(id)
	if err != nil { return err }
	return api.BookUploadFormat(book, uri)
}

func (api *API) BookExists(id uint64) (bool, error) {
	api.c.CheckRedirect = noRedirect
	resp, err := api.c.Head(fmt.Sprintf("%s/book/%d", api.url, id))
	api.c.CheckRedirect = http.DefaultClient.CheckRedirect
	if err != nil { return false, err }
	defer resp.Body.Close()

	if resp.StatusCode == 302 { return false, nil }
	return true, nil
}

func (api *API) DeleteBook(id uint64) error {
	// Check before deleting
	exists, err := api.BookExists(id)
	if err != nil { return err }
	if !exists { return errors.New("Book dosen't exist") }

	resp, err := api.c.Head(fmt.Sprintf("%s/delete/%d", api.url, id))
	if err != nil { return err }
	defer resp.Body.Close()
	return nil
}

func (api *API) DeleteBookFormat(id uint64, format Format) error {
	// Check before deleting
	exists, err := api.BookExists(id)
	if err != nil { return err }
	if !exists { return errors.New("Book dosen't exist") }

	resp, err := api.c.Head(fmt.Sprintf("%s/delete/%d/%s/", api.url, id, strings.ToUpper(format.Ext())))
	if err != nil { return err }
	defer resp.Body.Close()
	return nil
}

func (api *API) DownloadFormat(id uint64, format Format) (file *bytes.Buffer, filename string, err error) {
	resp, err := api.c.Get(fmt.Sprintf("%s/download/%d/%s/%d.%s", api.url, id, format.Ext(), id, format.Ext()))
	if err != nil { return nil, "", err }
	defer resp.Body.Close()
	if resp.StatusCode == 404 { return nil, "", ErrNotFound }

	filename, err = url.PathUnescape(strings.Split(resp.Header.Get("Content-Disposition"), "; ")[1][9:])
	if err != nil { return nil, "", err }
	file = new(bytes.Buffer)
	_, err = io.Copy(file, resp.Body)
	if err != nil { return nil, "", err }
	return file, filename, nil
}

func (api *API) DownloadCover(id uint64) (*bytes.Buffer, error) {
	resp, err := api.c.Get(fmt.Sprintf("%s/cover/%d", api.url, id))
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode == 404 { return nil, ErrNotFound }

	file := new(bytes.Buffer)
	_, err = io.Copy(file, resp.Body)
	if err != nil { return nil, err }
	return file, nil
}

func NewAPI(url string) (*API, error) {
	jar, err := cookiejar.New(nil)
	if err != nil { return nil, err }
	a := new(API)
	a.c = &http.Client{
		Jar: jar,
	}
	a.url = url
	return a, nil
}

