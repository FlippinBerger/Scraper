package main

import (
	"fmt"
	"os"
)

func main() {
	// Do some input checking
	if args := os.Args; len(args) != 2 {
		fmt.Println(fmt.Errorf("Expected a single command line argument, received: %d", len(args)-1))
		return
	}

	// Get the url to be scraped
	url := os.Args[1]

	scraper := NewScraper(url)

	err := scraper.Scrape()
	if err != nil {
		fmt.Printf("Scraping was unsuccessful due to: %s\n", err)
		return
	}

	// need to wait for all the writers to finish before we finish execution
	scraper.wg.Wait()

	err = scraper.WriteResults()
	if err != nil {
		fmt.Printf("Writing the results was unsuccessful due to: %s\n", err)
	}
}
