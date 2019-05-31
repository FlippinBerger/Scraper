package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
)

// Writer takes byte input on it's channel and writes the output to the
// defined resultsPath
type Writer struct {
	resultsPath string
	writer      chan []byte
}

// NewWriter constructs the Writer object to add your data to the file system
func NewWriter(path string) *Writer {
	w := new(Writer)

	var err error
	w.resultsPath, err = createResultsDirFor(path)
	if err != nil {
		log.Fatal(err)
	}

	w.writer = make(chan []byte)

	return w
}

// below here is all the work from the original Scraper class that needs to be
// cleaned up and made into a neat little writer

// getFriendlyString takes the entire link url, and changes it to
// Host + Path with the forward slashes replaced with underscores to be
// file system friendly
func getFriendlyString(fullLink string) string {
	// deal with poorly formed urls later if this is an issue
	url, _ := url.Parse(fullLink)

	// replace all forward slashes with underscores
	return strings.ReplaceAll(url.Host+url.Path, "/", "_")
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
