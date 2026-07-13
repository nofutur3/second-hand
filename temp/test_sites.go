package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	shops := []string{
		"https://www.bazos.cz",
		"https://www.sbazar.cz",
		"https://www.avizo.cz",
		"https://www.inzeruj.cz",
		"https://aukro.cz",
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for _, shop := range shops {
		fmt.Printf("Testing %s...\n", shop)

		req, err := http.NewRequest("GET", shop, nil)
		if err != nil {
			fmt.Printf("  ❌ Error creating request: %v\n\n", err)
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n\n", err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		fmt.Printf("  ✓ Status: %d\n", resp.StatusCode)
		fmt.Printf("  ✓ Body size: %d bytes\n", len(body))
		fmt.Printf("  ✓ Content-Type: %s\n\n", resp.Header.Get("Content-Type"))
	}
}
