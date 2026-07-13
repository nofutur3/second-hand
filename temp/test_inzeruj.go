package main

import (
	"fmt"
	"regexp"

	"github.com/gocolly/colly/v2"
)

func main() {
	fmt.Println("Testing Inzeruj...")

	c := colly.NewCollector()

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("Got response: %d bytes\n", len(r.Body))

		// Look for product URLs
		re := regexp.MustCompile(`/inzerat/([\w-]+)/([\w-]+)/([\w-]+)-(\d{6})\.html`)
		matches := re.FindAllStringSubmatch(string(r.Body), -1)
		fmt.Printf("Found %d matches\n", len(matches))

		for i, match := range matches {
			if i < 10 {
				fmt.Printf("  %d: %s (ID: %s)\n", i+1, match[0], match[4])
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Error: %v\n", err)
	})

	fmt.Println("Visiting...")
	err := c.Visit("https://www.inzeruj.cz/search?title=rypadlo")
	if err != nil {
		fmt.Printf("Visit error: %v\n", err)
	}

	fmt.Println("Waiting...")
	c.Wait()
	fmt.Println("Done!")
}
