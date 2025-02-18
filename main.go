package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"index/suffixarray"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	Limit = 20
)

func main() {
	searcher := Searcher{}
	err := searcher.Load("completeworks.txt")
	if err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/search", handleSearch(searcher))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	fmt.Printf("shakesearch available at http://localhost:%s...", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

type Searcher struct {
	CompleteWorks string
	SuffixArray   *suffixarray.Index
}

func handleSearch(searcher Searcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query, ok := r.URL.Query()["q"]
		if !ok || len(query[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing search query in URL params"))
			return
		}
		page := page(r.URL.Query().Get("p"))

		results := searcher.Search(query[0], page)
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		err := enc.Encode(results)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("encoding failure"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf.Bytes())
	}
}

func (s *Searcher) Load(filename string) error {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	s.CompleteWorks = string(dat)
	lowerWorks := strings.ToLower(s.CompleteWorks)
	s.SuffixArray = suffixarray.New([]byte(lowerWorks))
	return nil
}

func (s *Searcher) Search(query string, page int) []string {
	query = strings.ToLower(query)
	idxs := s.SuffixArray.Lookup([]byte(query), -1)
	offset := page * Limit
	if len(idxs) >= (offset + Limit) {
		idxs = idxs[offset : offset+Limit]
	} else if offset < len(idxs) {
		idxs = idxs[offset:]
	} else { // out of range, send empty
		idxs = []int{}
	}
	results := []string{}
	for _, idx := range idxs {
		results = append(results, s.CompleteWorks[idx-250:idx+250])
	}
	return results
}

func page(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	log.Printf("*> unexpected error occured when converting string %s to int on `page` sending zero\n", s)
	return 0
}
