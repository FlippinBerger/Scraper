# Scraper
A simple CLI web scraper written in Go.

The scraper takes a url as input, and will write the HTML of the page, plus every link on that page, to files an output directory.

If you've arrived here from Twitter, be sure to check out the non-master branch.  Currenty keeping master at the original implementation sans concurrency until I can get proper benchmarking tests done to see just how much performance I'm gaining from the new and improved branch that introduces Go concurrency constructs (channels and a wait group in this case).
