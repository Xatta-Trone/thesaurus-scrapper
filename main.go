package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/xatta-trone/thesaurus-scrapper/scrapper"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	p := os.Getenv("PORT")

	if p != "" {
		p = "8081"
	}

	PORT := fmt.Sprintf(":%s", p)

	r := gin.Default()
	r.Use(cors.Default())

	type KeyManager struct {
		Idx int
		Min int
		Max int
	}

	r.GET("/", func(c *gin.Context) {
		key := ""

		API_KEY := "key1,key2,key3,key4"

		splittedKeys := strings.Split(API_KEY, ",")
		keysLength := len(splittedKeys)

		if keysLength == 1 {
			key = API_KEY
		}

		if keysLength > 1 {

			keysRange := []KeyManager{}

			keysPerDay := int(math.Ceil(30.0 / float64(keysLength)))

			fmt.Println(keysPerDay)

			for i := 0; i < keysLength; i++ {
				max := (i + 1) * keysPerDay
				// set the max dey
				if i == keysLength-1 {
					max = 32
				}

				keyRange := KeyManager{
					Idx: i,
					Min: i*keysPerDay + 1,
					Max: max,
				}

				keysRange = append(keysRange, keyRange)

			}

			fmt.Println(keysRange)

			// now based on current date set the key
			// _, _, day := time.Now().UTC().Date()
			day := 12

			for _, keyRange := range keysRange {
				if day >= keyRange.Min && day <= keyRange.Max {
					key = splittedKeys[keyRange.Idx]
				}

			}

		}

		c.JSON(http.StatusOK, gin.H{
			"message": key,
		})
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.GET("/w/:word", func(c *gin.Context) {

		data, err := GetResult(c.Param("word"))

		fmt.Println(data, err)

		if err != nil && strings.Contains(strings.ToLower(err.Error()), "too many requests") {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}

		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}

		if len(data.Synonyms) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No data found"})
			return
		}

		c.JSON(200, gin.H{"data": data})
	})

	r.GET("/g/:word", func(c *gin.Context) {

		data, status := scrapper.GetGoogleResult(c.Param("word"))

		fmt.Println(data, status)

		if status != 200 {
			c.JSON(status, gin.H{"error": "could not find data"})
			return
		}

		c.JSON(200, gin.H{"data": data})
	})

	r.GET("/mw/:word", func(c *gin.Context) {

		data, err := GetMWData(c.Param("word"))

		fmt.Println(data, err)

		if err != nil && strings.Contains(strings.ToLower(err.Error()), "too many requests") {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}

		if err != nil && strings.Contains(strings.ToLower(err.Error()), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}

		if len(data.PartsOfSpeeches) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No data found"})
			return
		}

		c.JSON(200, gin.H{"data": data})
	})

	URL := ""

	if runtime.GOOS == "windows" {
		URL = "localhost" + PORT
	} else {
		URL = PORT
	}

	r.Run(URL)

	// GetMWData("abbess")

}

type WordResponse struct {
	Synonyms []Synonym `json:"synonyms"`
	Antonyms []string  `json:"antonyms"`
}

type Synonym struct {
	PartsOfSpeech string   `json:"parts_of_speech"`
	Definition    string   `json:"definition"`
	Syns          []string `json:"synonym"`
}

func GetResult(word string) (WordResponse, error) {

	var finalResult WordResponse
	var err error

	// temp PoS and Def
	tempPoS := []string{}
	tempDef := []string{}

	geziyor.NewGeziyor(&geziyor.Options{
		// StartRequestsFunc: func(g *geziyor.Geziyor) {
		// 	g.GetRendered("https://www.thesaurus.com/browse/"+word, g.Opt.ParseFunc)
		// },
		StartURLs: []string{"https://www.thesaurus.com/browse/" + word},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {

			if r.StatusCode != http.StatusOK {
				fmt.Println("There was an error, ", r.Status)
				err = fmt.Errorf("%s", r.Status)
			}

			// fmt.Println(string(r.Body))

			root := r.HTMLDoc.Find("[data-type='thesaurus-entry-module']")

			fmt.Println("roost")
			fmt.Println(root.Length())

			// find the parts of speech with definitions
			tabList := root.Find("[data-type='thesaurus-entry-tablist']")

			fmt.Println(tabList.Length())

			tabList.Find("li").Each(func(i int, s *goquery.Selection) {
				fmt.Println(s.Text())
				whole := s.Text()
				pos := s.Find("em").Text()
				def := strings.TrimLeft(strings.ReplaceAll(whole, pos, ""), " ")

				tempPoS = append(tempPoS, pos)
				tempDef = append(tempDef, def)

				fmt.Println(def)
				fmt.Println(pos)

			})

			singleGroup := []string{}

			card := root.Find("[data-type='thesaurus-synonyms-card']")

			card.Find("li").Each(func(i int, s *goquery.Selection) {
				fmt.Println(s.Text())
				sn := strings.TrimSpace(strings.ReplaceAll(s.Text(), "\n", " "))
				if len(sn) > 0 {
					singleGroup = append(singleGroup, sn)
				}
			})

			singleSynonymObj := Synonym{}

			if len(tempDef) > 0 {
				singleSynonymObj.Definition = tempDef[0]
				singleSynonymObj.PartsOfSpeech = tempPoS[0]
				singleSynonymObj.Syns = singleGroup
				finalResult.Synonyms = append(finalResult.Synonyms, singleSynonymObj)

			}

			// now find the antonyms
			antonyms := []string{}
			aCard := root.Find("[data-type='thesaurus-antonyms-card']")
			fmt.Println(aCard.Length())
			aCard.Find("li").Each(func(i int, s *goquery.Selection) {
				an := strings.TrimSpace(strings.ReplaceAll(s.Text(), "\n", " "))

				if len(an) > 0 {
					antonyms = append(antonyms, an)
				}
			})
			finalResult.Antonyms = antonyms
		},
		//BrowserEndpoint: "ws://localhost:3000",
	}).Start()

	return finalResult, err

	// Request the HTML page.
	res, err := http.Get("https://www.thesaurus.com/browse/" + word)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=========body==========")
	fmt.Println(res.Status)

	defer res.Body.Close()
	if res.StatusCode != 200 {

		return finalResult, errors.New(res.Status)
		// log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return finalResult, err
		// log.Fatal(err)
	}

	container := doc.Filter(".wjLcgFJpqs9M6QJsPf5v")

	fmt.Println(container.Length())

	// container := doc.Find(".MainContentContainer")

	// inside MainContentContainer
	// first ul parts of speech with definition
	// second ul synonyms
	// and followed by more synonyms for parts of speech
	// inside #antonyms the ul is the antonyms

	// check if definition is available or not
	defs := container.Find(".ew5makj1")
	// defs := container.Find("ul:first-child")

	if defs.Length() == 0 {
		fmt.Println("No definition available")
		return finalResult, nil
	}

	// not get the parts of speech
	defs.Each(func(i int, s *goquery.Selection) {
		// find parts of speech
		// fmt.Println("parts of speech", s.Find("em").Text())
		tempPoS = append(tempPoS, s.Find("em").Text())
		// fmt.Println("meaning", s.Find("strong").Text())
		tempDef = append(tempDef, s.Find("strong").Text())
	})

	// now find the synonyms and antonyms

	// len := container.Find("ul.e1ccqdb60").Length()
	// synonyms := container.Find("ul.e1ccqdb60").First().Find("li").Each(func(i int, s *goquery.Selection) {
	// 	fmt.Println(s.Find("a").Text())
	// })

	// synonyms := [][]string{}
	singleSynonymObj := Synonym{}

	// check if second synonym is available
	for i := 0; i < defs.Length(); i++ {
		singleGroup := []string{}
		container.Find("ul").Eq(i + 1).Find("li").Each(func(i int, s *goquery.Selection) {
			// fmt.Println(s.Find("a").Text())
			sn := strings.TrimSpace(strings.ReplaceAll(s.Find("a").Text(), "\n", " "))
			if len(sn) > 0 {
				singleGroup = append(singleGroup, sn)
			}

		})
		singleSynonymObj.Definition = tempDef[i]
		singleSynonymObj.PartsOfSpeech = tempPoS[i]
		singleSynonymObj.Syns = singleGroup

		finalResult.Synonyms = append(finalResult.Synonyms, singleSynonymObj)

		// synonyms = append(synonyms, singleGroup)
	}

	// fmt.Println(synonyms)

	antonyms := []string{}

	// find antonyms
	container.Find("#antonyms ul").Find("li").Each(func(i int, s *goquery.Selection) {
		// fmt.Println(s.Find("a").Text())
		// check string
		an := strings.TrimSpace(strings.ReplaceAll(s.Find("a").Text(), "\n", " "))

		if len(an) > 0 {
			antonyms = append(antonyms, an)
		}

	})

	finalResult.Antonyms = antonyms
	// fmt.Println(antonyms)

	return finalResult, nil

}
