package main

import (
	"fmt"
	"os"
)

// driver ideas:
// pagination of a website
// flag to set up the scraped content
// flag to run the scraped content locally
// flag to scrape and run

func main() {
	// Do some input checking
	if args := os.Args; len(args) != 2 {
		fmt.Println(fmt.Errorf("Expected a single command line argument, received: %d", len(args)-1))
		return
	}

	// Get the url to be scraped
	url := os.Args[1]

	writingFinished := make(chan bool)

	scraper := NewScraper(url, writingFinished)

	err := scraper.Scrape()
	if err != nil {
		fmt.Printf("Scraping was unsuccessful due to: %s\n", err)
		return
	}

	<-scraper.writer.finished

	// need to wait for all the writers to finish before we finish execution
	//scraper.wg.Wait()

	// May need to handle the other side of concurrency here to make 
	// sure the program isn't finishing too early
}
