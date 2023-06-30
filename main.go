package main

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/xatta-trone/thesaurus-scrapper/scrapper"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
		// log.Fatal("Error loading .env file")
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

		data, err := scrapper.GetResult(c.Param("word"))

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

		data, err := scrapper.GetMWData(c.Param("word"))

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

