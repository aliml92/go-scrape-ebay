package goscrapeebay

import (
	"bufio"
	"net/url"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
)

func (s *EbayScraper) scrapeLeafCateories(rootcategoryURL *url.URL, writer *bufio.Writer) error {
	s.logger.Info("Started scraping root category page", "url", rootcategoryURL.String())
	errChan := make(chan error, 1)

	c := colly.NewCollector(
		colly.CacheDir(s.config.CacheDir),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*.ebay.com",
		Delay:       s.config.Delay,
		RandomDelay: s.config.RandomDelay,
	})

	extensions.RandomUserAgent(c)

	c.OnRequest(func(r *colly.Request) {
		s.logger.Info("Visiting", "url", r.URL.String())
	})

	c.OnResponse(func(r *colly.Response) {
		s.logger.Info("Visited", "url", r.Request.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		s.logger.Error("Requesting Err",
			"url", r.Request.URL.String(),
			"statusCode", r.StatusCode,
			"err", err,
		)
		if r.StatusCode == 0 {
			select {
			case errChan <- ErrContextTimeout:
			default:
				// no op
			}
		}
	})

	c.OnHTML("body", func(h *colly.HTMLElement) {
		s.collectLeafCategoryURLs(c, h, writer)
	})

	c.OnScraped(func(r *colly.Response) {
		s.logger.Info("Scraped", "url", r.Request.URL.String())
	})

	go func() {
		err := c.Visit(rootcategoryURL.String())
		s.logger.Error("Visiting Err", "err", err)
		if err != nil {
			select {
			case errChan <- err:
			default:
				// no op
			}
		}
		close(errChan)
	}()

	err := <-errChan
	return err
}



func (s *EbayScraper) collectLeafCategoryURLs(
	c *colly.Collector,
	h *colly.HTMLElement,
	writer *bufio.Writer,
) {

	url := h.Request.URL.String()

	// check if the page is leaf category page
	if s.isLeafCategoryPage(h) {
		_, err := writer.WriteString(url + "\n")
		if err != nil {
			s.logger.Error("Failed to write to file", "err", err)
			return
		}
	} else {
		var count int
		h.ForEach("div.dialog__cell > section:first-of-type", func(i int, m *colly.HTMLElement) {
			title := m.ChildText("h2.section-title__title")
			if title == "" {
				s.logger.Warn("Carousel title empty", "url", url)
				return
			}
	
			if title == "Shop by Category" {
				m.ForEach("a.b-textlink", func(i int, d *colly.HTMLElement) {
					if !strings.Contains(d.Text, "See all") {
						if count > s.config.MaxCategoriesPerPage {
							return
						}
	
						url := d.Attr("href")
						err := c.Visit(url)
						if err != nil {
							s.logger.Error("Visiting Err", "err", err)
						}
						count++
					}
				})
	
			}
		})
	
		count = 0
		h.ForEach("section.brw-category-nav.brw-has-parentnode:first-of-type", func(i int, m *colly.HTMLElement) {
			title := m.ChildText("span.textual-display.brw-category-nav__title")
			if title == "" {
				s.logger.Warn("Carousel title empty", "url", url)
				return
			}
	
			if title == "Shop by Category" {
				m.ForEach("a.textual-display.brw-category-nav__link", func(i int, d *colly.HTMLElement) {
					if count > s.config.MaxCategoriesPerPage {
						return
					}
	
					url := d.Attr("href")
					err := c.Visit(url)
					if err != nil {
						s.logger.Error("Visiting Err", "err", err)
					}
					count++
				})
			}
		})
	}
}

func (s *EbayScraper) isLeafCategoryPage(h *colly.HTMLElement) bool {
	isLeaf := false
	url := h.Request.URL.String()
	
	checkTitle := func(i int, g *colly.HTMLElement) {
		title := g.ChildText("h2.section-title__title")
		if title == "" {
			s.logger.Warn("Carousel title empty", "url", url)
			return 
		}

		if title != "Shop by Category" && isRootPage(s.config.TargetURL, url) {
			isLeaf = true
		}
	}

	selectors := []string{
		"section.b-module.b-carousel.b-guidance.b-display--landscape:first-of-type",
		"section.b-module.b-carousel.b-guidance--text.b-display--landscape:first-of-type",
		"section.seo-guidance.seo-guidance__guidance_module:first-of-type",
		"section.b-module.b-visualnav:first-of-type",
		"section.brw-product-carousel:first-of-type",	
	}

	for _, selector := range selectors {
		h.ForEach(selector, checkTitle)
	}

	s.logger.Debug("Page type detected", "is_leaf", isLeaf, "url", url)
	
	return isLeaf
}

func isRootPage(root string, u string) bool {
	if root == u {
		return true
	}
	return false
}