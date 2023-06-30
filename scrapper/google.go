package scrapper

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// structs
type WordStruct struct {
	MainWord        string          `json:"word"`
	Audio           string          `json:"audio"`
	Phonetic        string          `json:"phonetic"`
	PartsOfSpeeches []PartsOfSpeech `json:"parts_of_speeches"`
}

type PartsOfSpeech struct {
	PartsOfSpeech string       `json:"parts_of_speech"`
	Phonetic      string       `json:"phonetic"`
	Audio         string       `json:"audio"`
	Definitions   []Definition `json:"definitions"`
}

type Definition struct {
	Definition string   `json:"definition"`
	Example    string   `json:"example"`
	Synonyms   []string `json:"synonyms"`
	Antonyms   []string `json:"antonyms"`
}

type ErrorResponse struct {
	Message string `json:"message" xml:"message"`
}

// constants
const (
	mainContainer        = "#center_col"
	jsSlotsFilterTag     = `div[jsslot=""]`
	mainWordQueryTag     = `span[data-dobid="hdw"]`
	mainWordAudioTag     = "audio"
	mainWordPhoneticsTag = "span.LTKOO"
	posDivFilterTag      = `div[jsname="r5Nvmf"]`
	posPhoneticsTag      = "span.LTKOO"
	posAudioTag          = "audio"
	posTag               = "span.YrbPuc"
	posEachDefinitionTag = `[data-dobid="dfn"]`
	posSynAntParentTag   = `div[role="list"]`
)

// regex
var IsLetter = regexp.MustCompile(`^[a-zA-Z\s-]+$`).MatchString

func GetGoogleResult(word string) (*WordStruct, int) {

	var wordS WordStruct
	errorStatus := 200

	// get env
	API_KEY := os.Getenv("SCRAPPER_API")

	if API_KEY == "" {
		fmt.Println("api key not found")
		return &wordS, 400
	}

	todaysKey := RoundRobinApiKey(API_KEY)

	// Request the HTML page.
	encoded_url := url.QueryEscape(fmt.Sprintf("https://www.google.com/search?&hl=en&q=define+%s",word))
	url := fmt.Sprintf("https://api.scrape.do?token=%s&url=%s", todaysKey, encoded_url)
	method := "GET"
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Println(err.Error())
		errorStatus = 400
		return &wordS, errorStatus
	}
	res, err := client.Do(req)
	// res, err := http.Get(fmt.Sprintf("https://api.scrape.do?token=%s&url=https://www.google.com/search?&hl=en&q=define+%s", todaysKey, word))
	if err != nil {
		fmt.Println(err.Error())
		errorStatus = 400
		return &wordS, errorStatus
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Println(res.Status)
		return &wordS, res.StatusCode
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Println(err.Error())
		return &wordS, res.StatusCode
		// log.Fatal(err)
	}

	root := doc.Find(mainContainer)

	if root.Length() == 0 {
		return &wordS, 404
	}

	// 	children 5 div with tag jsslot=""
	// 1. just the header
	// 2. search box
	// 3. all the definitions and the other things we want [our target]
	// 4. translations in other language
	// 5. use over time graph

	firstContainer := root.Find(".lr_container").First()

	fmt.Println("main container", root.Length())
	fmt.Println("firstContainer", firstContainer.Length())

	if firstContainer.Length() == 0 {
		return &wordS, 404
	}

	// 3rd div with attribute jsslot="" go obtain the main data
	thirdJsSlot := firstContainer.Find(jsSlotsFilterTag).FilterFunction(func(i int, s *goquery.Selection) bool {
		return i == 2
	})

	// 	inside the 3rd div 4 dive
	// 1- the main word
	// 2- see definitions in
	// 3- meanings [our target]
	// 4- origin

	// find the main word
	mainWord := strings.ReplaceAll(thirdJsSlot.Find(mainWordQueryTag).Text(),"·","")
	fmt.Println("main word", mainWord)
	// check if it has phonetics and audio in the #1 div

	firstDiv := thirdJsSlot.Children().First()

	mainWordAudio := firstDiv.Find(mainWordAudioTag).Children().AttrOr("src", "")
	mainWordPhonetics := firstDiv.Find(mainWordPhoneticsTag).First().Text()

	fmt.Println(mainWordAudio, mainWordPhonetics)
	wordS.MainWord = mainWord
	wordS.Audio = mainWordAudio
	wordS.Phonetic = mainWordPhonetics

	child := thirdJsSlot.Children()

	allPoses := []PartsOfSpeech{}

	// inside #3 div
	child.Find(posDivFilterTag).Each(func(i int, s *goquery.Selection) {
		// we are inside each parts of speech div
		poses := PartsOfSpeech{}
		// fmt.Println("=========================================================================")
		// fmt.Println(i, "th div")
		// get the phonetics
		phonetics := s.Find(posPhoneticsTag).First().Text()
		// fmt.Println("phonetics ::", phonetics)
		// get pronunciation the audio source
		audio := s.Find(posAudioTag).Children().AttrOr("src", "")
		// fmt.Println("audio ::", audio)

		// get the parts of speech
		pos := s.Find(posTag).First().Text()
		// fmt.Println("pos ::", pos)

		poses.Phonetic = phonetics
		poses.Audio = audio
		poses.PartsOfSpeech = pos

		// each meanings with examples
		s.Find("ol > li").Children().Each(func(i int, s *goquery.Selection) {
			// definition
			dfnElement := s.Find(posEachDefinitionTag)
			definition := Definition{}

			dfn := dfnElement.Text()

			if dfn != "" {
				fmt.Println("definition ::", dfn)
				// get the example sentence
				exElement := dfnElement.Siblings()
				example := strings.Trim(exElement.First().Text(), "\"")

				// fmt.Println("example ::", example)

				// now lets find the synonym and antonyms
				var synonyms []string
				var antonyms []string

				currentType := "Similar"
				synAntElements := s.Find(posSynAntParentTag)

				synAntElements.Children().Each(func(i int, s *goquery.Selection) {

					txtToAdd := strings.TrimSpace(s.Text())
					chkIfToAdd := false

					// filter out the grayed out words from the results
					if s.Children().First().AttrOr("style", "") == "cursor:text" {
						chkIfToAdd = false
					} else {
						chkIfToAdd = true
					}

					// omit first div with text h
					// now encounter Similar or Opposite

					if txtToAdd == "Similar:" {
						currentType = "synonyms"
					}
					if txtToAdd == "Opposite:" {
						currentType = "antonyms"
					}

					if currentType == "synonyms" && txtToAdd != "Similar:" && txtToAdd != "h" && txtToAdd != "" && chkIfToAdd {
						synonyms = append(synonyms, txtToAdd)
					}
					if currentType == "antonyms" && txtToAdd != "Opposite:" && txtToAdd != "h" && txtToAdd != "" && chkIfToAdd {
						antonyms = append(antonyms, txtToAdd)
					}

				})

				definition.Definition = dfn
				definition.Example = example
				definition.Synonyms = synonyms
				definition.Antonyms = antonyms

				// fmt.Println(currentType)

				// fmt.Println("synonyms ::", strings.Join(synonyms, ","))
				// fmt.Println("antonyms ::", strings.Join(antonyms, ","))
				poses.Definitions = append(poses.Definitions, definition)

			}
		})
		allPoses = append(allPoses, poses)

		// fmt.Println("=========================================================================")

	})

	wordS.PartsOfSpeeches = allPoses

	return &wordS, errorStatus
}

type KeyManager struct {
	Idx int
	Min int
	Max int
}

func RoundRobinApiKey(API_KEY string) string {
	key := ""
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

	fmt.Println(key)

	return key
}
