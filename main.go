package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve/v2"
	"log"
	"net/http"
	"os"
)

func main() {
	searcher := Searcher{indexPath: "shakesearch.bleve"}
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

	fmt.Printf("Listening on port %s...", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

type Searcher struct {
	indexPath string
}

func handleSearch(searcher Searcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query, ok := r.URL.Query()["q"]
		if !ok || len(query[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing search query in URL params"))
			return
		}
		results := searcher.Search(query[0])
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
	batchSize := 1000

	fmt.Println("loading file")
	index, err := bleve.Open(s.indexPath)
	defer index.Close()
	if err == bleve.ErrorIndexPathDoesNotExist {
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(s.indexPath, mapping)
		if err != nil {
			fmt.Println(err)
			fmt.Errorf("Create index: %w", err)
		}
		file, err := os.Open(filename)
		if err != nil {
			fmt.Errorf("Open file: %w", err)
		}
		defer file.Close()

		batch := index.NewBatch()
		batchCount := 0

		// Start reading from the file using a scanner.
		scanner := bufio.NewScanner(file)
		linecount := 0
		for scanner.Scan() {
			line := scanner.Text()
			batch.Index(fmt.Sprintf("line: %d. %s", linecount, line), line)
			batchCount++

			if batchCount >= batchSize {
				err = index.Batch(batch)
				if err != nil {
					return err
				}
				batch = index.NewBatch()
				batchCount = 0
			}

			linecount += 1
			fmt.Println(linecount)
		}
		if batchCount > 0 {
			err = index.Batch(batch)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	return nil
}

func (s *Searcher) Search(query string) []string {
	fmt.Println("search for: ", query)
	results := []string{}
	index, err := bleve.Open(s.indexPath)
	defer index.Close()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	q := bleve.NewFuzzyQuery(query)
	search := bleve.NewSearchRequest(q)
	sr, err := index.Search(search)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	rv := fmt.Sprintf("%d matches, showing %d through %d, took %s\n", sr.Total, sr.Request.From+1, sr.Request.From+len(sr.Hits), sr.Took)
	results = append(results, rv)

	for i, hit := range sr.Hits {
		results = append(results, fmt.Sprintf("%5d. %s (%f)\n", i+sr.Request.From+1, hit.ID, hit.Score))
	}
	return results
}
