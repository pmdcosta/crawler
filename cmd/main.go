package main

import (
	"net/url"
	"strings"
)

func main() {

}

// Scrape the document's content to gather all links
func processLinks(doc *goquery.Document) (result []*url.URL) {
	baseURL, _ := doc.FindMatcher(baseHrefMatcher).Attr("href")
	urls := doc.FindMatcher(aHrefMatcher).Map(func(_ int, s *goquery.Selection) string {
		val, _ := s.Attr("href")
		if baseURL != "" {
			val = handleBaseTag(doc.Url, baseURL, val)
		}
		return val
	})
	for _, s := range urls {
		// If href starts with "#", then it points to this same exact URL, ignore (will fail to parse anyway)
		if len(s) > 0 && !strings.HasPrefix(s, "#") {
			if parsed, e := url.Parse(s); e == nil {
				parsed = doc.Url.ResolveReference(parsed)
				result = append(result, parsed)
			} else {
				w.logFunc(LogIgnored, "ignore on unparsable policy %s: %s", s, e.Error())
			}
		}
	}
	return
}