package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// Scraper does the, you guessed it, scraping
type Scraper struct {
	// client to take care of all our HTTP requests
	client    *http.Client

	// original URL to be downloaded locally
	targetURL string

	// channel that will accept the links as they are encountered
	// and will then process them as they come in
	pages chan string

	// the writer takes care of adding the scrapers results to the file system
	// in a clean manner
	writer *Writer

	wg *sync.WaitGroup
}

// NewScraper is a contructor that will give us our Scraper object from a url
func NewScraper(target string, writerFinished chan bool) *Scraper {
	scraper := new(Scraper)
	scraper.targetURL = target

	// use a custom http client that has a 10s timeout to prevent hanging
	// for too long on any request
	scraper.client = &http.Client{
		Timeout: time.Second * 30,
	}

	scraper.pages = make(chan string)
	scraper.writer = NewWriter(target)
	scraper.wg = new(sync.WaitGroup)

	return scraper
}

// fetchURL is a helper method that returns the byte array for a given webpage
func (s *Scraper) fetchURL(target string) ([]byte, error) {
	fmt.Println("fetching url for ", target)
	response, err := s.client.Get(target)

	fmt.Println("Got the response back for ", target)

	if err != nil {
		fmt.Println("Couldn't do it 1: ", err)
		return nil, err
	}
	// close
	defer response.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Couldn't do it 2: ", err)
		return nil, err
	}

	fmt.Println("Made it all the way through; returning data")

	return data, nil
}

// localizeLink changes the href in the link in the original parent doc
// to point to the local file so that we can later browse the entire page
// offline
func localizeLink(data []byte, link string) {
	return	
}

// handleLinks will find all the links in byte slice and save them off into
// a string slice. This will later be enhanced to be a bit more concurrent,
// probably taking in a channel and sending the links to that channel to be
// processed as they're encountered. This will greatly improve efficiency
// because this method has to walk through the entirety of the html tree
// to find all the links
func (s *Scraper) handleLinks(data []byte) {
	reader := bytes.NewReader(data)
	// Tokenize the HTML
	tokenizer := html.NewTokenizer(reader)

	for {
		// If the HTML has ended, we break out of the loop
		token := tokenizer.Next()

		if token == html.ErrorToken {
			fmt.Println(tokenizer.Err())
			break
		}

		// New Token started
		if token == html.StartTagToken {
			fmt.Println("found a start token")
			// Check if the token is an <a> tag
			if name, _ := tokenizer.TagName(); string(name) == "a" {
				for {
					// Get the next attribute
					name, val, more := tokenizer.TagAttr()

					// Check if the attribute is "href"
					if string(name) == "href" {
						fmt.Printf("Adding %s to the pages channel\n", string(val))
						s.pages <- string(val)
					}

					// There are no more attributes so we break out of the
					// attribute search loop.
					if !more {
						break
					}
				}
			}
		}
	}

	// close the pages channel, because we're done sending links to it
	fmt.Println("Closing the pages channel")
	close(s.pages)

	return
}

// Scrape starts at the url, and sends the results to the writer
// and also kicks off a goroutine to do the same for each link that it finds along the way
func (s *Scraper) Scrape() error {
	response, err := s.client.Get(s.targetURL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// data was fetched cleanly for this url
	var data []byte
	data, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	} 

	// replace all forward slashes with underscores
	fsFriendlyStr := getFriendlyString(s.targetURL)

	// we made it through and got our first page's data, so it's safe 
	// and proper to start accepting data on our writer's channel
	go s.writer.AcceptData()

	// send results of the main piece to the writer channel
	linkData := &LinkData{path: fsFriendlyStr, data: data}
	s.writer.writer <- linkData

	// parse the links out of the byte slice of the main page's data
	go s.handleLinks(data)

	// go through the pages channel until it is closed and empty
	for link := range s.pages {
		s.wg.Add(1)
		go s.scraperHelper(link)
	}

	// finished scraping all the links on the main page, 
	// so we are safe to close our writer now as well
	fmt.Println("closing the writer chanel")

	s.wg.Wait()
	close(s.writer.writer)

	return nil
}

// scraperHelper will make a request for each link encountered, and send the
// data to the writer. It will not follow any more links to restrict the scope
func (s *Scraper) scraperHelper(target string) {
	defer s.wg.Done()
	fmt.Printf("Located a child link at %s, starting to scrape it too.\n", target)

	// get the data
	data, err := s.fetchURL(target)
	if err != nil {
		fmt.Printf("Couldn't get data for %s due to %s\n", target, err)
		return
	}

	// since we're only going one level deep, we just need to send this data
	// to the writer
	target = getFriendlyString(target)
	fmt.Printf("Sending data for %s to the writer.", target)

	// send results of this page to the writer channel
	linkData := &LinkData{path: target, data: data}
	s.writer.writer <- linkData
}