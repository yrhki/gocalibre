package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/yrhki/gocalibre/calibre-web"
)

var (
	flagURL      string
	flagUsername string
	flagPassword string
	flagLogin    bool
)

func must(err error, reason string, callback func()) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error while", reason)
		if callback != nil { callback() }
		exitMessage(err)
	}
}

func exitMessage(text interface{}) {
	fmt.Fprintln(os.Stderr, text)
	os.Exit(1)
}

func parseArgs() {
	flag.StringVar(&flagUsername, "username", "", "username for calibre")
	flag.StringVar(&flagPassword, "password", "", "username for calibre")
	flag.StringVar(&flagURL, "url", "", "")
	flag.Parse()

	if v := os.Getenv("CALIBRE_PASSWORD"); v != "" { flagPassword = v }
	if v := os.Getenv("CALIBRE_USERNAME"); v != "" { flagUsername = v }
	if v := os.Getenv("CALIBRE_URL"); v != "" { flagURL = v }

	if flagURL == "" { must(errors.New("empty URL"), "parsing arguments", nil) }

	return
}

func main() {
	parseArgs()
	api, err := calibre.NewAPI(flagURL)
	must(err, "creating api instance", nil)

	must(api.Login(flagUsername, flagPassword), "login in", nil)
	defer api.Logout()

	switch flag.Arg(0) {
	case "list":
		switch flag.Arg(1) {
		case "book":
			listBook(api)
		case "lang":
			listLang(api)
		case "categories":
			listCategories(api)
		case "series":
			listSeries(api)
		case "authors":
			listAuthors(api)
		default:
			exitMessage("usage: clibrecli list <book|lang|categories|series|authors>")
		}
	case "delete":
		if flag.Arg(1) == "" { exitMessage("usage: clibrecli delete <BOOKID>") }

		id, err := strconv.ParseUint(flag.Arg(1), 10, 0)
		must(err, "parsing BOOKID", nil)

		if prompt(false, "Delete book") { deleteBook(api, id) }
	case "download":
		if flag.Arg(1) == "" { exitMessage("usage: clibrecli download <URL>") }

		// TODO
	case "upload":
		if flag.Arg(1) == "" { exitMessage("usage: clibrecli upload <FILPATH> [FILEPATH..]") }

		b, err := uploadFile(api, flag.Arg(1), flag.Args()[2:])
		must(err, "uploading book", func() {
			if prompt(true, fmt.Sprintf("Delete book: %s (%d)", b.Title, b.ID())) {
				deleteBook(api, b.ID())
			}
		})
	}
}
