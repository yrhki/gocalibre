package calibre

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/PuerkitoBio/goquery"
)

var (
	ErrNotFound = errors.New("format not found")
)

func noRedirect(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func downloadFormat(id uint64, format Format) string {
	return fmt.Sprintf("/download/%d/%s/%d.%s", id, format.Ext(), id, format.Ext())
}

func checkFlashAlert(resp *http.Response) error {
	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil { return err }

	if flash := doc.Find("#flash_alert"); len(flash.Nodes) > 0 {
		return errors.New(flash.Text())
	}

	if flash := doc.Find("#flash_warning"); len(flash.Nodes) > 0 {
		fmt.Fprintln(os.Stderr, "WARN:", flash.Text())
	}

	return nil
}
