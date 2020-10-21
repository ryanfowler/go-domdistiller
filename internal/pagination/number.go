// ORIGINAL: java/PageParameterParser.java

package pagination

import (
	nurl "net/url"
	"strconv"
	"strings"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-domdistiller/internal/domutil"
	"github.com/markusmobius/go-domdistiller/internal/model"
	"github.com/markusmobius/go-domdistiller/internal/pagination/info"
	"github.com/markusmobius/go-domdistiller/internal/pagination/parser"
	"github.com/markusmobius/go-domdistiller/internal/stringutil"
	"golang.org/x/net/html"
)

// PageNumberFinder parses the document to collect groups of adjacent plain text numbers and
// outlinks with digital anchor text.
type PageNumberFinder struct {
	wordCounter              stringutil.WordCounter
	timingInfo               *model.TimingInfo
	adjacentNumberGroups     *info.MonotonicPageInfoGroups
	numForwardLinksProcessed int
}

func NewPageNumberFinder(wc stringutil.WordCounter, timingInfo *model.TimingInfo) *PageNumberFinder {
	return &PageNumberFinder{
		wordCounter:          wc,
		timingInfo:           timingInfo,
		adjacentNumberGroups: &info.MonotonicPageInfoGroups{},
	}
}

// FindOutlink parses the document to collect outlinks with numeric anchor text and numeric text
// around them. Returns PageParamInfo, always (never null). If no page parameter is detected or
// determined to be best, its Type is info.Unset.
func (pnf *PageNumberFinder) FindOutlink(root *html.Node, pageURL *nurl.URL) *info.PageParamInfo {
	idx := 0
	allLinks := dom.GetElementsByTagName(root, "a")
	for idx < len(allLinks) {
		link := allLinks[idx]
		pageInfo, _ := pnf.getPageInfoAndText(link, pageURL)
		if pageInfo == nil {
			idx++
			continue
		}

		// This link is a good candidate for pagination.

		// Close current group of adjacent numbers, add a new group if necessary.
		pnf.adjacentNumberGroups.AddGroup()

		// Before we append the link to the new group of adjacent numbers, check if it's
		// preceded by a text node with numeric text; if so, add it before the link.
		pnf.findAndAddClosestValidLeafNodes(link, false, true, pageURL)

		// Add the link to the current group of adjacent numbers.
		pnf.adjacentNumberGroups.AddPageInfo(pageInfo)

		// Add all following text nodes and links with numeric text.
		pnf.numForwardLinksProcessed = 0
		pnf.findAndAddClosestValidLeafNodes(link, false, false, pageURL)

		// Skip the current link and links already processed in the forward
		// findandAddClosestValidLeafNodes().
		idx += 1 + pnf.numForwardLinksProcessed
	}

	pnf.adjacentNumberGroups.CleanUp()
	paramInfo := parser.DetectParamInfo(pnf.adjacentNumberGroups, pageURL.String())
	return paramInfo
}

// getPageInfoAndText returns a populated PageInfoAndText if given link is to be added to
// adjacentNumbersGroups. Otherwise, returns null if link is to be ignored. "javascript:"
// links with numeric text are considered valid links to be added.
func (pnf *PageNumberFinder) getPageInfoAndText(link *html.Node, pageURL *nurl.URL) (*info.PageInfo, string) {
	// In original dom-distiller they ignore the invisible link. Unfortyunately it's
	// impossible to do that here. NEED-COMPUTE-CSS.

	// Get visible text using innerText instead of textContent
	linkText := strings.TrimSpace(domutil.InnerText(link))
	number, err := pnf.linkTextToNumber(linkText)
	if err != nil || number < 0 || number > MaxNumForPageParam {
		return nil, ""
	}

	linkHref := dom.GetAttribute(link, "href")
	linkHref = stringutil.CreateAbsoluteURL(linkHref, pageURL)

	isEmptyHref := linkHref == ""
	isJavascriptLink := strings.HasPrefix(linkHref, "javascript:")

	var hrefURL *nurl.URL
	if !isEmptyHref && !isJavascriptLink {
		hrefURL, err = nurl.ParseRequestURI(linkHref)
		if err != nil || hrefURL.Host != pageURL.Host {
			return nil, ""
		}

		hrefURL, _ = nurl.Parse(linkHref)
		hrefURL.Path = strings.TrimSuffix(hrefURL.Path, "/")
		hrefURL.RawPath = hrefURL.Path
		hrefURL.Fragment = ""
		hrefURL.RawFragment = ""
	}

	if isEmptyHref || isJavascriptLink {
		return &info.PageInfo{
			PageNumber: number,
			URL:        linkHref,
		}, linkText
	}

	return &info.PageInfo{
		PageNumber: number,
		URL:        hrefURL.String(),
	}, linkText
}

// findAndAddClosestValidLeafNodes finds and adds the leaf node(s) closest to the given start node.
// This recurses and keeps finding and, if necessary, adding the numeric text of valid nodes, collecting
// the PageInfo for the current adjacency group. For backward search, i.e. nodes before start node,
// search terminates (i.e. recursion stops) once a text node or anchor is encountered. If the text
// node contains numeric text, it's added to the current adjacency group. Otherwise, a new group is
// created to break the adjacency.
// For forward search, i.e. nodes after start node, search continues (i.e. recursion continues) until
// a text node or anchor with non-numeric text is encountered. In the process, text nodes and anchors
// with numeric text are added to the current adjacency group. When a non-numeric text node or anchor
// is encountered, a new group is started to break the adjacency, and search ends.
//
// Returns true to continue search, false to stop.
func (pnf *PageNumberFinder) findAndAddClosestValidLeafNodes(start *html.Node, checkStart, backward bool, pageURL *nurl.URL) bool {
	var node *html.Node
	if checkStart {
		node = start
	} else {
		if backward {
			node = start.PrevSibling
		} else {
			node = start.NextSibling
		}
	}

	if node == nil {
		node = start.Parent
		if rxInvalidParentWrapper.MatchString(domutil.NodeName(node)) {
			return false
		}
		return pnf.findAndAddClosestValidLeafNodes(node, false, backward, pageURL)
	}

	checkStart = false
	switch node.Type {
	case html.TextNode:
		// Text must contain words.
		text := node.Data
		if text == "" || pnf.wordCounter.Count(text) == 0 {
			break
		}

		added := pnf.addNonLinkTextIfValid(text)

		// For backward search, we're done regardless if text was added.
		// For forward search, we're done only if text was invalid, otherwise continue.
		if backward || !added {
			return false
		}

	case html.ElementNode:
		if dom.TagName(node) == "a" {
			// For backward search, we're done because we've already processed the anchor.
			if backward {
				return false
			}

			// For forward search, we're done only if link was invalid, otherwise continue.
			pnf.numForwardLinksProcessed++
			added := pnf.addLinkIfValid(node, pageURL)
			if !added {
				return false
			}
			break
		}

		// Intentionally fallthrough
		fallthrough

	default:
		// Check children nodes.
		if len(dom.ChildNodes(node)) == 0 {
			break
		}

		checkStart = true // We want to check the child node.
		if backward {
			// Start the backward search with the rightmost child i.e. last and closest to
			// given node.
			node = node.LastChild
		} else {
			// Start the forward search with the leftmost child i.e. first and closest to
			// given node.
			node = node.FirstChild
		}
	}

	return pnf.findAndAddClosestValidLeafNodes(node, checkStart, backward, pageURL)
}

// addNonLinkTextIfValid handles the text for a non-link node. Each numeric term in the text
// that is a valid plain page number adds a PageParamInfo.PageInfo into the current adjacent
// group. All other terms break the adjacency in the current group, adding a new group instead.
//
// Returns true if text was added to current group of adjacent numbers. Otherwise, false with
// a new group created to break the current adjacency.
func (pnf *PageNumberFinder) addNonLinkTextIfValid(text string) bool {
	// If the text does not contain valid number(s); if necessary, current group of adjacent
	// numbers should be closed, adding a new group if possible.
	if !rxNumber.MatchString(text) {
		pnf.adjacentNumberGroups.AddGroup()
		return false
	}

	// Extract terms from the text, differentiating between those that contain only digits and
	// those that contain non-digits.
	added := false
	for _, term := range rxTerms.FindAllString(text, -1) {
		number := -1
		termWithDigits := rxSurroundingDigits.FindStringSubmatch(term)
		if len(termWithDigits) > 1 {
			number, _ = strconv.Atoi(termWithDigits[1])
		}

		if number >= 0 && number <= MaxNumForPageParam {
			// This text is a valid candidate of plain text page number, add it to last group of
			// adjacent numbers.
			pnf.adjacentNumberGroups.AddNumber(number, "")
			added = true
		} else {
			// The text is not a valid number, so current group of adjacent numbers should be
			// closed, adding a new group if possible.
			pnf.adjacentNumberGroups.AddGroup()
		}
	}

	return added
}

// addLinkIfValid adds PageInfo to the current adjacent group for a link if its text is numeric.
// Otherwise, add a new group to break the adjacency.
//
// Returns true if link was added, false otherwise.
func (pnf *PageNumberFinder) addLinkIfValid(link *html.Node, pageURL *nurl.URL) bool {
	pageInfo, _ := pnf.getPageInfoAndText(link, pageURL)
	if pageInfo != nil {
		pnf.adjacentNumberGroups.AddPageInfo(pageInfo)
		return true
	}

	pnf.adjacentNumberGroups.AddGroup()
	return false
}

func (pnf *PageNumberFinder) linkTextToNumber(linkText string) (int, error) {
	linkText = rxLinkNumberCleaner.ReplaceAllString(linkText, "")
	linkText = strings.TrimSpace(linkText)
	return strconv.Atoi(linkText)
}