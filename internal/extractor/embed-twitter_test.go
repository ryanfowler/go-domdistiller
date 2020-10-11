// ORIGINAL: javatest/EmbedExtractorTest.java

package extractor_test

import (
	"testing"

	"github.com/alecthomas/assert"
	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-domdistiller/internal/extractor"
	"github.com/markusmobius/go-domdistiller/internal/testutil"
	"github.com/markusmobius/go-domdistiller/internal/webdoc"
)

func Test_Extractor_Twitter_ExtractNotRenderedBasic(t *testing.T) {
	tweetBlock := dom.CreateElement("blockquote")
	dom.SetAttribute(tweetBlock, "class", "twitter-tweet")

	p := dom.CreateElement("p")
	dom.AppendChild(p, testutil.CreateAnchor("http://twitter.com/foo", "extra content"))
	dom.AppendChild(tweetBlock, p)
	dom.AppendChild(tweetBlock, testutil.CreateAnchor("http://twitter.com/foo/bar/12345", "January 1, 1900"))

	extractor := extractor.NewTwitterExtractor()
	result, _ := (extractor.Extract(tweetBlock)).(*webdoc.Embed)

	assert.NotNil(t, result)
	assert.Equal(t, "twitter", result.Type)
	assert.Equal(t, "12345", result.ID)

	// Test trailing slash
	tweetBlock = dom.CreateElement("blockquote")
	dom.SetAttribute(tweetBlock, "class", "twitter-tweet")

	p = dom.CreateElement("p")
	dom.AppendChild(p, testutil.CreateAnchor("http://twitter.com/foo", "extra content"))
	dom.AppendChild(tweetBlock, p)
	dom.AppendChild(tweetBlock, testutil.CreateAnchor("http://twitter.com/foo/bar/12345///", "January 1, 1900"))

	result, _ = (extractor.Extract(tweetBlock)).(*webdoc.Embed)

	assert.NotNil(t, result)
	assert.Equal(t, "twitter", result.Type)
	assert.Equal(t, "12345", result.ID)
}

func Test_Extractor_Twitter_ExtractNotRenderedTrailingSlash(t *testing.T) {
	tweetBlock := dom.CreateElement("blockquote")
	dom.SetAttribute(tweetBlock, "class", "twitter-tweet")

	p := dom.CreateElement("p")
	dom.AppendChild(p, testutil.CreateAnchor("http://twitter.com/foo", "extra content"))
	dom.AppendChild(tweetBlock, p)
	dom.AppendChild(tweetBlock, testutil.CreateAnchor("http://twitter.com/foo/bar/12345///", "January 1, 1900"))

	extractor := extractor.NewTwitterExtractor()
	result, _ := (extractor.Extract(tweetBlock)).(*webdoc.Embed)

	assert.NotNil(t, result)
	assert.Equal(t, "twitter", result.Type)
	assert.Equal(t, "12345", result.ID)
}

func Test_Extractor_Twitter_ExtractNotRenderedBadTweet(t *testing.T) {
	tweetBlock := dom.CreateElement("blockquote")
	dom.SetAttribute(tweetBlock, "class", "random-class")

	p := dom.CreateElement("p")
	dom.AppendChild(p, testutil.CreateAnchor("http://nottwitter.com/foo", "extra content"))
	dom.AppendChild(tweetBlock, p)
	dom.AppendChild(tweetBlock, testutil.CreateAnchor("http://nottwitter.com/12345", "timestamp"))

	extractor := extractor.NewTwitterExtractor()
	result, _ := (extractor.Extract(tweetBlock)).(*webdoc.Embed)

	assert.Nil(t, result)
}

func Test_Extractor_Twitter_ExtractRenderedBasic(t *testing.T) {
	tweet := dom.CreateElement("iframe")
	dom.SetAttribute(tweet, "id", "twitter-widget")
	dom.SetAttribute(tweet, "title", "Twitter Tweet")
	dom.SetAttribute(tweet, "src", "https://platform.twitter.com/embed/index.html")
	dom.SetAttribute(tweet, "data-tweet-id", "12345")

	extractor := extractor.NewTwitterExtractor()
	result, _ := (extractor.Extract(tweet)).(*webdoc.Embed)

	assert.NotNil(t, result)
	assert.Equal(t, "twitter", result.Type)
	assert.Equal(t, "12345", result.ID)
}

func Test_Extractor_Twitter_ExtractRenderedBadTweet(t *testing.T) {
	tweet := dom.CreateElement("iframe")
	dom.SetAttribute(tweet, "id", "twitter-widget")
	dom.SetAttribute(tweet, "title", "Twitter Tweet")
	dom.SetAttribute(tweet, "src", "https://platform.not-twitter.com/embed/index.html")
	dom.SetAttribute(tweet, "data-bad-id", "12345")

	extractor := extractor.NewTwitterExtractor()
	result, _ := (extractor.Extract(tweet)).(*webdoc.Embed)

	assert.Nil(t, result)
}