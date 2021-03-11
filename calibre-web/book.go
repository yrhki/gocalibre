package calibre

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"mime/multipart"
	"strings"
	"time"
)

type Format uint8

func (f Format) Ext() string {
	switch f {
	case FormatPDF:
		return "pdf"
	case FormatEPUB:
		return "epub"
	case FormatMOBI:
		return "mobi"
	case FormatAZW3:
		return "azw3"
	case FormatDOCX:
		return "docx"
	case FormatRTF:
		return "rtf"
	case FormatFB2:
		return "fb2"
	case FormatLIT:
		return "lit"
	case FormatLRF:
		return "lrf"
	case FormatTXT:
		return "txt"
	case FormatHTMLZ:
		return "htmlz"
	case FormatODT:
		return "odt"
	case FormatCBZ:
		return "cbz"
	default:
		panic(fmt.Sprintf("unhandled format %d", f))
	}
}



const (
	FormatPDF Format = iota
	FormatMOBI
	FormatEPUB
	FormatAZW3
	FormatDOCX
	FormatRTF
	FormatFB2
	FormatLIT
	FormatLRF
	FormatTXT
	FormatHTMLZ
	FormatODT
	FormatCBZ
)


type ListBook struct {
	id uint64
	name string
	authors []Author
}

func (b *ListBook) Name() string { return b.name }
func (b *ListBook) ID() uint64 { return b.id }
func (b *ListBook) Authors() []Author { return b.authors }

type Rating uint8

const (
	RatingNone Rating = iota
	Rating1
	Rating2
	Rating3
	Rating4
	Rating5
)

func GetRating(rating uint8) Rating {
	switch rating {
	case 0:
		return RatingNone
	case 1:
		return Rating1
	case 2:
		return Rating2
	case 3:
		return Rating3
	case 4:
		return Rating4
	case 5:
		return Rating5
	default:
		return Rating5
	}
}

type Book struct {
	id uint64
	formats map[Format]bool

	Title string
	Series string
	Rating uint8
	SeriesIndex float64
	Published *time.Time
	Description string
	Authors []string
	Categories []string
	Publisher string
	Languages []string
	Identifiers BookIdentifiers
}

func (book *Book) ID() uint64 { return book.id }

func (book *Book) HasFormat(format Format) bool {
	t, ok := book.formats[format]
	return ok && t
}

func (book *Book) multipart() (*multipart.Writer, *bytes.Buffer, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	err := w.WriteField("book_title", book.Title)
	if err != nil { return nil, nil, err }
	err = w.WriteField("author_name", strings.Join(book.Authors, " & "))
	if err != nil { return nil, nil, err }
	err = w.WriteField("description", book.Description)
	if err != nil { return nil, nil, err }
	err = w.WriteField("tags", strings.Join(book.Categories, ", "))
	if err != nil { return nil, nil, err }
	err = w.WriteField("series", book.Series)
	if err != nil { return nil, nil, err }
	err = w.WriteField("series_index", fmt.Sprintf("%v", book.SeriesIndex))
	if err != nil { return nil, nil, err }
	err = w.WriteField("rating", fmt.Sprintf("%d", book.Rating))
	if err != nil { return nil, nil, err }
	err = w.WriteField("cover_url", "")
	if err != nil { return nil, nil, err }

	if book.Published != nil {
		err = w.WriteField("pubdate", book.Published.Format("2006-01-02"))
		if err != nil { return nil, nil, err }
	} else {
		err = w.WriteField("pubdate", "")
		if err != nil { return nil, nil, err }
	}

	err = w.WriteField("publisher", book.Publisher)
	if err != nil { return nil, nil, err }
	err = w.WriteField("languages", strings.Join(book.Languages, ", "))
	if err != nil { return nil, nil, err }

	for t, v := range book.Identifiers {
		id := uint(math.Floor(rand.Float64() * 1000000))

		err = w.WriteField(fmt.Sprintf("identifier-type-%d", id), t)
		if err != nil { return nil, nil, err }
		err = w.WriteField(fmt.Sprintf("identifier-val-%d", id), v)
		if err != nil { return nil, nil, err }
	}
	return w, &b, nil
}



type BookIdentifiers map[string]string

func (iden BookIdentifiers) hasIdentifier(t string) (string, bool) {
	if value, ok := iden[t]; ok {
		return value, ok
	} else {
		return "", ok
	}
}

func (iden BookIdentifiers) ISBN() (string, bool) { return iden.hasIdentifier("isbn") }
func (iden BookIdentifiers) Amazon() (string, bool) { return iden.hasIdentifier("amazon") }
func (iden BookIdentifiers) DOI() (string, bool) { return iden.hasIdentifier("doi") }
func (iden BookIdentifiers) Douban() (string, bool) { return iden.hasIdentifier("douban") }
func (iden BookIdentifiers) Goodreads() (string, bool) { return iden.hasIdentifier("goodreads") }
func (iden BookIdentifiers) Google() (string, bool) { return iden.hasIdentifier("google") }
func (iden BookIdentifiers) Kobo() (string, bool) { return iden.hasIdentifier("kobo") }
func (iden BookIdentifiers) ISSN() (string, bool) { return iden.hasIdentifier("issn") }
func (iden BookIdentifiers) ISFDB() (string, bool) { return iden.hasIdentifier("isfdb") }
func (iden BookIdentifiers) Lubimyczytac() (string, bool) { return iden.hasIdentifier("lubimyczytac") }
func (iden BookIdentifiers) URL() (string, bool) { return iden.hasIdentifier("url") }




