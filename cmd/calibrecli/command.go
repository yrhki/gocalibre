package main

import (
	"fmt"

	"github.com/yrhki/gocalibre/calibre-web"
	"github.com/yrhki/gocalibre/cmd/calibrecli/upload"
)



func listBook(api *calibre.API) {
	books, err := api.ListBooks()
	must(err, "loading books", nil)

	l := fmt.Sprint(len(fmt.Sprint(len(books))))

	for _, book := range books {
		fmt.Printf("%0" + l + "d: %s\n", book.ID(), book.Name())
	}
}

func listLang(api *calibre.API) {
	list, err := api.GetLanguages()
	must(err, "loading languages", nil)

	for _, item := range list { fmt.Println(item) }
}

func listCategories(api *calibre.API) {
	list, err := api.GetCategories()
	must(err, "loading categories", nil)

	for _, item := range list { fmt.Println(item) }
}

func listSeries(api *calibre.API) {
	list, err := api.GetSeries()
	must(err, "loading series", nil)

	for _, item := range list { fmt.Println(item) }
}

func listAuthors(api *calibre.API) {
	list, err := api.GetAuthors()
	must(err, "loading authors", nil)

	for _, item := range list { fmt.Println(item) }
}

func deleteBook(api *calibre.API, id uint64) {
	err := api.DeleteBook(id)
	must(err, "deleting book", nil)
}

func downloadBook(api *calibre.API, url string) {
	err := upload.Download(api, url)
	must(err, "downloading book", nil)
}

func uploadFile(api *calibre.API, uri string, formats []string) (*calibre.Book, error) {
	book, err := api.Upload(uri)
	if err != nil { return nil, err }
	fmt.Println("Uploaded book:", book.Title)

	for _, file := range formats {
		err = api.BookUploadFormat(book, file)
		if err != nil { return book, err }
		fmt.Println("Uploaded format:", file)
	}


	return book, nil
}
