package scrapper

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

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
	// tempPoS := []string{}
	// tempDef := []string{}
	StartURLs := "https://www.thesaurus.com/browse/" + word

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", true))...)
	defer cancel()
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// run task list

	err = chromedp.Run(ctx,
		chromedp.Navigate(StartURLs),
		// chromedp.WaitVisible("body"),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		fmt.Println(err)
		return finalResult, err
	}

	checkRootXpath := "/html/body/div[1]/div/main/div[2]/div[2]/div[1]/section/div/h1"

	// Execute JavaScript in the browser context to get total number of elements matching the XPath expression
	var checkRoot int
	err = chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`(function() {
            var elements = document.evaluate('%s', document, null, XPathResult.ANY_TYPE, null);
            var length = 0;
            while (elements.iterateNext()) {
                length++;
            }
            return length;
        })()`, checkRootXpath), &checkRoot))

	if err != nil {
		fmt.Println(err)
		return finalResult, err
	}

	if checkRoot == 0 {
		fmt.Println("Root not found")
		return finalResult, nil
	}

	// check total parts of speech
	totalPOSXpath := "/html/body/div[1]/div/main/div[2]/div[2]/div[2]/section/div[@data-type=\"synonym-and-antonym-card\"]"

	// Execute JavaScript in the browser context to get total number of elements matching the XPath expression
	var totalPOSLength int
	err = chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`(function() {
            var elements = document.evaluate('%s', document, null, XPathResult.ANY_TYPE, null);
            var length = 0;
            while (elements.iterateNext()) {
                length++;
            }
            return length;
        })()`, totalPOSXpath), &totalPOSLength))

	if err != nil {
		fmt.Println("No synonym found")
		return finalResult, nil
	}

	if totalPOSLength == 0 {
		fmt.Println("No synonym found")
		return finalResult, nil
	}

	fmt.Println("Total POS length:", totalPOSLength)

	for i := 0; i < totalPOSLength; i++ {
		var currentPos Synonym
		var syns []string
		// iterate over all the POS
		synonymRoot := fmt.Sprintf("/html/body/div[1]/div/main/div[2]/div[2]/div[2]/section/div[@data-type=\"synonym-and-antonym-card\"][%v]/div[2]/div[2]/div", i+1)

		// Execute JavaScript in the browser context to get total number of elements matching the XPath expression
		var totalSynonymLength int
		err = chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`(function() {
            var elements = document.evaluate('%s', document, null, XPathResult.ANY_TYPE, null);
            var length = 0;
            while (elements.iterateNext()) {
                length++;
            }
            return length;
        })()`, synonymRoot), &totalSynonymLength))

		if err != nil {
			fmt.Println(err)
			continue
		}

		if totalSynonymLength == 0 {
			fmt.Println("No synonym found")
			return finalResult, err
		}

		fmt.Println("Total synonym length:", totalSynonymLength)

		posXpath := fmt.Sprintf(" /html/body/div[1]/div/main/div[2]/div[2]/div[2]/section/div[@data-type=\"synonym-and-antonym-card\"][%v]/div[1]/p", i+1)
		// get the pos and definition
		var posString string
		_ = chromedp.Run(ctx,
			chromedp.Evaluate(fmt.Sprintf(`document.evaluate('%s',document,null,XPathResult.FIRST_ORDERED_NODE_TYPE,null,).singleNodeValue?.textContent`, posXpath), &posString))

		poss := strings.Split(posString, " as in ")

		if len(poss) == 2 {
			currentPos.Definition = strings.TrimSpace(poss[1])
			currentPos.PartsOfSpeech = poss[0]
		}

		fmt.Println(posString)

		for j := 0; j < totalSynonymLength; j++ {
			// Define your XPath expression
			xpathExpression := fmt.Sprintf("/html/body/div[1]/div/main/div[2]/div[2]/div[2]/section/div[@data-type=\"synonym-and-antonym-card\"][%v]/div[2]/div[2]/div[%v]/ul/li", i+1, j+1)

			// Check if the XPath expression is valid
			var isValid bool
			err = chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`(function() {
            try {
                document.evaluate('%s', document, null, XPathResult.ANY_TYPE, null);
                return true;
            } catch (e) {
                return false;
            }
        })()`, xpathExpression), &isValid))

			if err != nil {
				fmt.Println(err)
				continue
			}

			if !isValid {
				log.Printf("Invalid XPath expression")
				continue
			}

			var nodes []interface{}
			err = chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`(function() {
            var nodes = [];
            var elements = document.evaluate('%s', document, null, XPathResult.ORDERED_NODE_SNAPSHOT_TYPE, null);
            for (var i = 0; i < elements.snapshotLength; i++) {
                nodes.push(elements.snapshotItem(i).textContent.trim());
            }
            return nodes;
        })()`, xpathExpression), &nodes))

			if err != nil {
				fmt.Println(err)
				continue
			}

			// Convert interface{} slice to []string
			for _, node := range nodes {
				syns = append(syns, node.(string))
			}

			fmt.Println(syns)

		}
		currentPos.Syns = append(currentPos.Syns, syns...)
		finalResult.Synonyms = append(finalResult.Synonyms, currentPos)

	}

	if len(finalResult.Synonyms) > 0 {
		finalResult.Antonyms = append(finalResult.Antonyms, "")
	}

	return finalResult, err

}
