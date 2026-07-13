package main

import (
	"fmt"
	"strings"

	"github.com/gocolly/colly/v2"
)

func main() {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"),
	)

	url := "https://www.bazos.cz/search.php?hledat=hemingway"

	fmt.Printf("Fetching: %s\n\n", url)

	// Try to find any tables
	c.OnHTML("table", func(e *colly.HTMLElement) {
		class := e.Attr("class")
		if class != "" {
			fmt.Printf("Found table with class: %s\n", class)
		}
	})

	// Try to find divs with inzerat
	c.OnHTML("[class*='inzerat']", func(e *colly.HTMLElement) {
		fmt.Printf("Found element: %s with class: %s\n", e.Name, e.Attr("class"))
		// Print first few children
		e.ForEach("*", func(i int, child *colly.HTMLElement) {
			if i < 3 {
				fmt.Printf("  Child %d: %s (class: %s)\n", i, child.Name, child.Attr("class"))
			}
		})
	})

	// Look for any h2 or h3 tags (likely titles)
	c.OnHTML("h2, h3", func(e *colly.HTMLElement) {
		text := strings.TrimSpace(e.Text)
		if text != "" && len(text) < 100 {
			fmt.Printf("Found heading: %s (class: %s)\n", text, e.Attr("class"))
		}
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("\nGot response: %d bytes\n\n", len(r.Body))
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Error: %v\n", err)
	})

	c.Visit(url)
	c.Wait()
}
