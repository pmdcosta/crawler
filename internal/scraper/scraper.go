package scraper

import (
	"bytes"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseHref = "a[href]"
	attrHref = "href"
)

// ScrapePage returns all the links in a html page
func ScrapePage(root *url.URL, page []byte) map[string]int {
	// load the HTML document
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if err != nil {
		return nil
	}
	return scrapeDocument(root, doc)
}

// scrapeDocument returns all the links in a html document
func scrapeDocument(root *url.URL, doc *goquery.Document) map[string]int {
	var urls = make(map[string]int)
	doc.Find(baseHref).Each(func(_ int, sel *goquery.Selection) {
		// process each url found
		href, exists := sel.Attr(attrHref)
		if !exists {
			return
		}
		if u := processHref(root, href); u != nil {
			urls[u.String()] += 1
		}
	})
	return urls
}

// processHref processes the reference and returns the url
func processHref(root *url.URL, href string) *url.URL {
	// ignore # which points to the exact same URL that is being processed
	if href == "" || strings.HasPrefix(href, "#") {
		return nil
	}

	// enhance the url with the base path if missing
	href, err := addBasePath(root, href)
	if err != nil {
		return nil
	}

	// parse the reference
	parsed, err := url.Parse(href)
	if err != nil {
		return nil
	}

	return root.ResolveReference(parsed)
}

// addBasePath adds the base path if it is missing in the url
func addBasePath(root *url.URL, href string) (string, error) {
	hrefURL, err := root.Parse(href)
	if err != nil {
		return "", err
	}
	return hrefURL.String(), nil
}
