package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// Scraper does the, you guessed it, scraping
type Scraper struct {
	client    *http.Client
	targetURL string
	links     []string
	results   map[string][]byte
	wg        *sync.WaitGroup
}

// NewScraper is a contructor that will give us our Scraper object from a url
func NewScraper(target string) *Scraper {
	scraper := new(Scraper)

	// use a custom http client that has a 10s timeout to prevent hanging
	// for too long on any request
	scraper.client = &http.Client{
		Timeout: time.Second * 10,
	}

	scraper.targetURL = target
	scraper.results = make(map[string][]byte)

	scraper.wg = new(sync.WaitGroup)

	return scraper
}

// fetchURL is a helper method that returns the byte array for a given webpage
func (s *Scraper) fetchURL(target string) ([]byte, error) {
	response, err := s.client.Get(target)

	if err != nil {
		return nil, err
	}
	// close
	defer response.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// getLinks will find all the links in byte slice and save them off into
// a string slice. This will later be enhanced to be a bit more concurrent,
// probably taking in a channel and sending the links to that channel to be
// processed as they're encountered. This will greatly improve efficiency
// because this method has to walk through the entirety of the html tree
// to find all the links
func getLinks(data []byte) []string {
	reader := bytes.NewReader(data)
	// Tokenize the HTML
	tokenizer := html.NewTokenizer(reader)

	// this won't be a thing anymore once we're concurrent
	var links []string

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
						// this will be the addition to the links channel
						// once that exists
						// Cast Url
						links = append(links, string(val))
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

	return links
}

// Scrape starts at the url, collects its html into the results map,
// and also kicks off a goroutine to do the same for each link that it finds along the way
func (s *Scraper) Scrape() error {
	response, err := s.client.Get(s.targetURL)
	if err != nil {
		return err
	}
	// close
	defer response.Body.Close()

	// data was fetched cleanly for this url, add to results before parsing
	var data []byte
	data, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	} else {
		fmt.Printf("Size of data is really %d\n", len(data))
	}

	// replace all forward slashes with underscores
	fsFriendlyStr := getFriendlyString(s.targetURL)

	s.results[fsFriendlyStr] = data

	// get links will just be called instead of returning a value it'll use a channel
	links := getLinks(data)

	// right here is where we'd read from the channel until it's done

	// Parse the data for links and kick off scraperHelpers for each
	// going to need to tell our driver how many links it needs to wait for
	// in terms of concurrency
	for _, link := range links {
		s.wg.Add(1)
		go s.scraperHelper(link)
	}

	return nil
}

// scraperHelper will make a request for each link encountered, and write to
// the results map. It will not follow any more link to restrict the scope
func (s *Scraper) scraperHelper(target string) error {
	fmt.Printf("Located a child link at %s, starting to scrape it too.\n", target)
	defer s.wg.Done()

	// get the data
	data, err := s.fetchURL(target)
	if err != nil {
		return err
	}

	// since we're only going one level deep, we just need to add this url
	// and its data to the results map
	fmt.Printf("Adding byte data to the results map for %s\n", target)
	target = getFriendlyString(target)

	// results map is going to become a call to the writer to actually
	// write the data
	s.results[target] = data
	return nil
}

// createResultsDirFor will create a directory at the PWD with the name
// targetURL_results. It'll wipe the dir if it already exists
func createResultsDirFor(target string) (string, error) {
	target = getFriendlyString(target)
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	resultsDirName := target + "_results"
	resultsDirPath := pwd + "/" + resultsDirName

	// if the results Dir exists we need to delete it
	if _, err := os.Stat(resultsDirPath); !os.IsNotExist(err) {
		os.RemoveAll(resultsDirPath)
	}

	os.Mkdir(resultsDirPath, 0777)

	return resultsDirPath, nil
}

// writeFile will write the file to the path given
func writeFile(path, target string, data []byte) error {
	target = getFriendlyString(target)
	pathWithFile := path + "/" + target

	_, err := os.Create(pathWithFile)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(pathWithFile, data, 0666)
	if err != nil {
		return err
	}

	return nil
}

// WriteResults will transfer each entry in the results map to a file in the FS
func (s *Scraper) WriteResults() error {
	fmt.Println("Writing the results to output folder.")

	// Write the original target url first:
	friendlyStr := getFriendlyString(s.targetURL)
	data, exists := s.results[friendlyStr]
	if !exists {
		return fmt.Errorf("The target URL %s wasn't in the result set", s.targetURL)
	}

	// We have data to write, create the directory under targetURL_results if
	// it doesn't already exist
	path, err := createResultsDirFor(s.targetURL)
	if err != nil {
		return fmt.Errorf("Unable to create results dir for %s with %s", s.targetURL, err)
	}

	err = writeFile(path, s.targetURL, data)
	if err != nil {
		return fmt.Errorf("Unable to write file for %s with %s", s.targetURL, err)
	}

	// write the original target url to a file named target_targetURL

	// loop through all the keys in the result set and write them to their own
	// files as long as they aren't the parent
	for k := range s.results {
		fmt.Printf("k is %s\n", k)
		if k == s.targetURL {
			continue
		}

		// write this urls data to a file named k
		err = writeFile(path, k, s.results[k])
		if err != nil {
			return fmt.Errorf("Unable to write file for %s with %s", k, err)
		}
	}

	return nil
}
