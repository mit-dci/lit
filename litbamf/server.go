package litbamf

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
)

func BamfListen(port uint16) {

	listenString := fmt.Sprintf("http://127.0.0.1:%d", port)
	exPath := path.Dir(os.Args[0])

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		jsExpr, _ := regexp.Compile("/js.*")
		imagesExpr, _ := regexp.Compile("/images.*")
		styleExpr, _ := regexp.Compile("/style.*")

		path := []byte(r.URL.Path)

		if r.URL.Path == "/ws" {
			return
		} else if jsExpr.Match(path) || imagesExpr.Match(path) || styleExpr.Match(path) {
		} else {
			r.URL.Path = "/"
		}

		http.FileServer(http.Dir(exPath+"/litbamf/public")).ServeHTTP(w, r)
	})
	go http.ListenAndServe(listenString, nil)
}
