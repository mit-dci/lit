package litbamf

import (
	"fmt"
	"net/http"
	"regexp"
)

func BamfListen(bamfport uint16, litHomeDir string) {
	listenString := fmt.Sprintf("http://127.0.0.1:%d", bamfport)

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

		http.FileServer(http.Dir(litHomeDir+"/litbamf/")).ServeHTTP(w, r)
	})
	go http.ListenAndServe(listenString, nil)
}
