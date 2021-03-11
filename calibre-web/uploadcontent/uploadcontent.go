package uploadcontent

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mitchellh/ioprogress"
)

type Content interface {
	Size() int64
	ContentType() string
	Filename() string
	Close() error
	Reader() io.Reader
}

type content struct {
	r io.ReadCloser
	size int64
	contentType string
	filename string
	verbose bool
	uri string
}

func (c *content) Close() error { return c.r.Close() }
func (c *content) ContentType() string { return c.contentType }
func (c *content) Filename() string { return c.filename }
func (c *content) Size() int64 { return c.size }

func (c *content) Reader() io.Reader {
	if c.verbose {
		return &ioprogress.Reader{
			Reader: c.r,
			Size: c.size,
			DrawFunc: ioprogress.DrawTerminalf(os.Stdout, func(progress, total int64) string {
				return fmt.Sprintf("Uploading [%s]: %s", ioprogress.DrawTextFormatBytes(progress, total), c.uri)
			}),
		}
	} else {
		return c.r
	}
}

func ContentFromResponse(resp *http.Response, verbose bool) Content {
	resp.Request.URL.RawQuery = ""
	return &content{
		r: resp.Body,
		size: resp.ContentLength,
		contentType: resp.Header.Get("Content-Type"),
		filename: filepath.Base(resp.Request.URL.Path),
		uri: resp.Request.URL.Redacted(),
		verbose: verbose,
	}
}

func ContentFromFile(file *os.File, verbose bool) (Content, error) {
	// Detect ContentType
	b := make([]byte, 512)
	_, err := file.Read(b)
	if err != nil { return nil, err }
	contentType := http.DetectContentType(b)

	// Reset position
	_, err = file.Seek(0, 0)
	if err != nil { return nil, err }

	s, err := file.Stat()
	if err != nil { return nil, err }

	return &content{
		r: file,
		size: s.Size(),
		contentType: contentType,
		filename: filepath.Base(file.Name()),
		uri: file.Name(),
		verbose: verbose,
	}, nil
}
