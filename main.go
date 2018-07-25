package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func main() {
	var urlFlag = flag.String("url", "https://golang.org", "URL of the root web page to build sitemap from")
	flag.Parse()
	visitedLinks := map[string]bool{*urlFlag: true}
	fmt.Println("Starting with root page: " + *urlFlag)
	links := parsedLinksFromURL(*urlFlag)
	for len(links) > 0 {
		if _, ok := visitedLinks[links[0].String()]; !ok {
			fmt.Println("Visiting URL: " + links[0].String())
			links = append(links, parsedLinksFromURL(links[0].String())...)
			visitedLinks[links[0].String()] = true
		}
		links = links[1:]
	}
}

func parsedLinksFromURL(URL string) []*url.URL {
	resp, err := http.Get(URL)
	var result []*url.URL
	if err != nil {
		fmt.Printf("Unsuccessful GET for: %s\n", URL)
		// ignore links with GET failure
		return result
	}
	defer resp.Body.Close()
	doc, err := html.Parse(resp.Body)
	if err != nil {
		panic(err)
	}
	for l := range Links(doc) {
		rootURL := getParsedURL(URL)
		candidateURL, err := url.Parse(l.Href)
		if err != nil {
			// ignore if unable to parse URL
			continue
		}
		if isLinkToFollow(candidateURL, rootURL) {
			candidateURL.Scheme = rootURL.Scheme
			candidateURL.Host = rootURL.Host
			result = append(result, candidateURL)
		}
	}
	return result
}

func isLinkToFollow(candidateURL *url.URL, rootURL *url.URL) bool {
	// Don't follow references
	if candidateURL.Fragment != "" {
		return false
	}
	// Follow if no domain specified (starts with '/')
	if candidateURL.Host == "" {
		return true
	}
	// Follow if the link url is in the same domain as the root url
	if domainsMatch(rootURL, candidateURL) {
		return true
	}
	return false
}

func getParsedURL(URL string) *url.URL {
	parsedURL, err := url.Parse(URL)
	if err != nil {
		panic(err)
	}
	return parsedURL
}

func domainsMatch(u1 *url.URL, u2 *url.URL) bool {
	if withoutWWW(u1.Host) == withoutWWW(u2.Host) {
		return true
	}
	return false
}

func withoutWWW(host string) string {
	result := host
	if strings.HasPrefix(host, "www") {
		result = result[4:]
	}
	return result
}

// Link represents a link (<a> href="..."> in an HTML
// document.
type Link struct {
	Href string
	Text string
}

// Links returns the links contained in an html node.
func Links(n *html.Node) <-chan Link {
	c := make(chan Link)
	go func() {
		defer close(c)
		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "a" {
				for _, a := range n.Attr {
					if a.Key == "href" {
						c <- Link{a.Val, text(n)}
					}
				}
			}
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				f(child)
			}
		}
		f(n)
	}()
	return c
}

// text will return a concatenated string of the text nodes below
// the node passed in.
func text(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	// do not recurse for not elementnodes
	if n.Type != html.ElementNode {
		return ""
	}
	var ret string
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		ret += text(child)
	}
	return strings.Join(strings.Fields(ret), " ")
}
