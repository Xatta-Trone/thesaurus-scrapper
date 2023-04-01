package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type MWResult struct {
	Word            string            `json:"word"`
	PartsOfSpeeches []PartsOfSpeeches `json:"parts_of_speeches"`
}

type PartsOfSpeeches struct {
	PartsOfSpeech string `json:"parts_of_speech"`
	Data          []Data `json:"data"`
}

type Data struct {
	AsIn       string   `json:"as_in"`
	Definition string   `json:"definition"`
	Example    string   `json:"example"`
	Synonyms   []string `json:"synonyms"`
	Antonyms   []string `json:"antonyms"`
}

func GetMWData(word string) (MWResult, error) {

	var result MWResult

	// Request the HTML page.
	res, err := http.Get("https://www.merriam-webster.com/thesaurus/" + word)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Println(res.Status)
		return result, errors.New(res.Status)
		// log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Println(err)
		return result, err
		// log.Fatal(err)
	}

	container := doc.Find("#left-content")

	// find the divs with class entry-word-section-container

	eachPoS := container.Find(".entry-word-section-container")

	if eachPoS.Length() == 0 {
		fmt.Println("No data available")
		return result, nil
	}

	fmt.Println(eachPoS.Children().Find(".thes-word-list-item").Length())

	if eachPoS.Children().Find(".thes-word-list-item").Length() == 0 {
		fmt.Println("No data available")
		return result, nil

	}

	eachPoS.Each(func(i int, s *goquery.Selection) {

		var eachPos PartsOfSpeeches

		// find the parts of speech
		pos := s.Find(".parts-of-speech").Text()
		// fmt.Println(pos)

		eachPos.PartsOfSpeech = strings.TrimSpace(pos)

		// now go for each as in words

		s.Find(".vg-sseq-entry-item").Each(func(i int, g *goquery.Selection) {
			var data Data
			// as in word
			asIn := g.Find(".as-in-word").Text()
			// fmt.Println(asIn)
			data.AsIn = strings.TrimSpace(asIn)

			if g.Find(".dt").Length() > 0 {
				// definition
				def := g.Find(".dt").Get(0).FirstChild.Data
				// example
				ex := g.Find(".dt").Children().Text()

				// fmt.Println(strings.TrimSpace(def))
				// fmt.Println(strings.TrimSpace(ex))
				data.Definition = strings.TrimSpace(def)
				data.Example = strings.TrimSpace(ex)

			}

			// synonyms
			synonyms := []string{}
			antonyms := []string{}

			// get lists
			lists := g.Find(".synonyms_list")

			// first item is synonyms
			// second item is antonym

			if lists.Length() > 1 {
				lists.First().Find(".thes-word-list-item").Each(func(i int, s *goquery.Selection) {
					synonyms = append(synonyms, strings.TrimSpace(s.Text()))
				})
				lists.Eq(1).Find(".thes-word-list-item").Each(func(i int, s *goquery.Selection) {
					antonyms = append(antonyms, strings.TrimSpace(s.Text()))
				})
			}

			// fmt.Println(synonyms)
			// fmt.Println(antonyms)

			data.Synonyms = synonyms
			data.Antonyms = antonyms

			eachPos.Data = append(eachPos.Data, data)

		})

		result.PartsOfSpeeches = append(result.PartsOfSpeeches, eachPos)
		result.Word = word

	})

	// d, _ := json.MarshalIndent(result, "", "\t")

	// fmt.Println(string(d))

	return result, nil

}
