package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
)

// LinkData houses all the logic to write a scraped webpage to the filesystem
type LinkData struct {
	path string
	data []byte
}

// Writer takes byte input on it's channel and writes the output to the
// defined resultsPath
type Writer struct {
	resultsPath string
	writer      chan *LinkData
}

// NewWriter constructs the Writer object to add your data to the file system
func NewWriter(path string) *Writer {
	w := new(Writer)

	var err error
	w.resultsPath, err = createResultsDirFor(path)
	if err != nil {
		log.Fatal(err)
	}

	w.writer = make(chan *LinkData)

	return w
}

// getFriendlyString takes the entire link url, and changes it to
// Host + Path with the forward slashes replaced with underscores to be
// file system friendly
func getFriendlyString(fullLink string) string {
	// deal with poorly formed urls later if this is an issue
	url, _ := url.Parse(fullLink)

	// replace all forward slashes with underscores
	return strings.Replace(url.Host+url.Path, "/", "_", -1)
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

	// if the results Dir exists; delete it
	// TODO can be smarter here in the future
	if _, err := os.Stat(resultsDirPath); !os.IsNotExist(err) {
		os.RemoveAll(resultsDirPath)
	}

	os.Mkdir(resultsDirPath, 0777)

	return resultsDirPath, nil
}

// AcceptData watches the writer's channel for new data to write 
// to the file system
func (w *Writer) AcceptData() {
	fmt.Printf("size of writer loop is: %d\n", len(w.writer))
	for linkData := range w.writer {
		fmt.Println("writing to ", linkData.path)
		w.Write(linkData)
	}
	fmt.Println("finishing the AcceptData loop")
}

// Write will parse out the linkData and write it to the filesystem
func (w *Writer) Write(linkData *LinkData) {
	fsFriendlyName := getFriendlyString(linkData.path)
	err := w.writeFile(fsFriendlyName, linkData.data)

	if err != nil {
		fmt.Printf("Writing of %s was unsuccessful due to: %s\n", fsFriendlyName, err)
	}
}

// writeFile will write the file to the path given
func (w *Writer) writeFile(target string, data []byte) error {
	target = getFriendlyString(target)
	pathWithFile := w.resultsPath + "/" + target

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