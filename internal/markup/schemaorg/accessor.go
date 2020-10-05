// ORIGINAL: java/SchemaOrgParserAccessor.java

package schemaorg

import (
	"github.com/markusmobius/go-domdistiller/internal/model"
)

func (ps *Parser) Title() string {
	// In original dom-distiller they sort the articles by their area in
	// descending order. Unfortunately it's not possible in Go, so we
	// don't do that here. NEED-COMPUTE-CSS.
	for _, item := range ps.getArticleItems() {
		title := item.getStringProperty(HeadlineProp)
		if title == "" {
			title = item.getStringProperty(NameProp)
		}

		if title != "" {
			return title
		}
	}

	return ""
}

func (ps *Parser) Type() string {
	// Returns Article if there's an article.
	if len(ps.getArticleItems()) > 0 {
		return "Article"
	}

	return ""
}

func (ps *Parser) URL() string {
	articles := ps.getArticleItems()
	if len(articles) > 0 {
		return articles[0].getStringProperty(URLProp)
	}

	return ""
}

func (ps *Parser) Images() []model.MarkupImage {
	// Images are ordered as follows:
	// 1) the "associatedMedia" or "encoding" image of the article that first declares it,
	// 2) or the first ImageObject with "representativeOfPage" as "true",
	// 3) then, the list of "image" property of remaining articles,
	// 4) lastly, the list of ImageObject's.
	images := []model.MarkupImage{}

	// First, get images from ArticleItem's.
	var associatedImageOfArticle *ImageItem
	for _, item := range ps.getArticleItems() {
		// If this is the first article with an associated image, remember it for now;
		// it'll be added to the list later when its position in the list can be determined.
		if associatedImageOfArticle == nil {
			associatedImageOfArticle = item.getRepresentativeImageItem()
			if associatedImageOfArticle != nil {
				continue
			}
		}

		image := item.getImage()
		if image != nil {
			images = append(images, *image)
		}
	}

	// Then, get images from ImageItem's.
	hasRepresentativeImage := false
	for _, item := range ps.getImageItems() {
		image := item.getImage()

		// Insert `image` at beginning of list if it's the associated image of an
		// article, or it's the first image that's representative of page.
		if item == associatedImageOfArticle || (!hasRepresentativeImage && item.isRepresentativeOfPage()) {
			hasRepresentativeImage = true
			images = append([]model.MarkupImage{*image}, images...)
		} else {
			images = append(images, *image)
		}
	}

	return images
}

func (ps *Parser) Description() string {
	articles := ps.getArticleItems()
	if len(articles) > 0 {
		return articles[0].getStringProperty(DescriptionProp)
	}

	return ""
}

func (ps *Parser) Publisher() string {
	// Returns either the "publisher" or "copyrightHolder" property
	// of the first article.
	var publisher string

	if articles := ps.getArticleItems(); len(articles) > 0 {
		publisher = articles[0].getPersonOrOrganizationName(PublisherProp)
		if publisher == "" {
			publisher = articles[0].getPersonOrOrganizationName(CopyrightHolderProp)
		}
	}

	return publisher
}

func (ps *Parser) Copyright() string {
	articles := ps.getArticleItems()
	if len(articles) > 0 {
		return articles[0].getCopyright()
	}

	return ""
}

func (ps *Parser) Author() string {
	var author string

	if articles := ps.getArticleItems(); len(articles) > 0 {
		author = articles[0].getPersonOrOrganizationName(AuthorProp)
		// If there's no "author" property, use "creator" property
		if author == "" {
			author = articles[0].getPersonOrOrganizationName(CreatorProp)
		}
	}

	// Otherwise, use "rel=author" tag.
	if author == "" {
		author = ps.authorFromRel
	}

	return author
}

func (ps *Parser) Article() *model.MarkupArticle {
	articles := ps.getArticleItems()
	if len(articles) == 0 {
		return nil
	}

	return articles[0].getArticle()
}

func (ps *Parser) OptOut() bool {
	return false
}