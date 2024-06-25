package goscrapeebay

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
)

type Product struct {
	Categories []string            `json:"categories"`
	Name       string              `json:"name"`
	Price      string              `json:"price"`
	Available  string              `json:"available"`
	Sold       string              `json:"sold"`
	ImageLinks []string            `json:"image_links"` // TODO: see if it is possible to reference image to product variant
	Specs      map[string]string   `json:"specs"`       // TODO: use ordered map to retain product specs order
	Attributes map[string][]string `json:"attributes"`

	// TODO: collect more data if necessary
	// product URL, ebay item number, seller information, ...
}


func (s *EbayScraper) setupProductDetailsCollector(w *bufio.Writer, errChan chan error) *colly.Collector {
	c := colly.NewCollector(
		colly.CacheDir(s.config.CacheDir),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*.ebay.com",
		Delay:       s.config.Delay,
		RandomDelay: s.config.RandomDelay,
	})

	extensions.RandomUserAgent(c)

	detailsCollector := c.Clone()

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
		var c int
		h.ForEach("a.s-item__link", func(i int, f *colly.HTMLElement) {
			if c < s.config.MaxProductsPerPage {
				link := f.Attr("href")
				link = strings.Split(link, "?")[0]
				s.logger.Debug("product url found", "url", link)
				detailsCollector.Visit(link)
				
				c++
			}
		})

		c = 0 
		h.ForEach("a.bsig__title__wrapper", func(i int, f *colly.HTMLElement) {
			link := f.Attr("href")
			link = strings.Split(link, "?")[0]
			s.logger.Debug("product url found", "url", link)
			if strings.Contains(link, "ebay.com/itm/") &&  c < s.config.MaxProductsPerPage {
				detailsCollector.Visit(link)
				c++
			}
		})		
	})

	detailsCollector.OnRequest(func(r *colly.Request) {
		s.logger.Info("Visiting", "url", r.URL.String())
	})

	detailsCollector.OnHTML(".vim.x-vi-evo-main-container.template-evo-avip", func(h *colly.HTMLElement) {
		s.scrapeDetails(h, w)
	})

	c.OnScraped(func(r *colly.Response) {
		s.logger.Info("Scraped", "url", r.Request.URL.String())
	})

	return c
}


func (s *EbayScraper) scrapeDetails(e *colly.HTMLElement, writer *bufio.Writer) {
	s.logger.Debug("Scraping product details...")
	product := Product{}

	// Step 1. Scrape category breadcrumb
	var categories []string
	e.ForEach("nav.breadcrumbs li > a > span", func(_ int, h *colly.HTMLElement) {
		categories = append(categories, h.Text)
	})

	if len(categories) == 0 {
		fmt.Println("Failed to scrape product categories")
		return
	}

	product.Categories = categories
	// Step 2. Scrape product name
	productName := e.ChildText("h1.x-item-title__mainTitle > span.ux-textspans.ux-textspans--BOLD")
	if productName == "" {
		fmt.Println("Failed to scrape product name")
		return
	}

	product.Name = productName

	// Step 3. Scrape product prices
	price := e.ChildText("div.x-price-primary > span")

	if price == "" {
		fmt.Println("Failed to scrape product price")
		return
	}

	product.Price = price

	// Step 4. Scrape available and sold counts
	available := e.ChildText("div.d-quantity__availability > div > span:first-child")
	sold := e.ChildText("div.d-quantity__availability > div > span:last-child")

	product.Available = available
	product.Sold = sold

	// Step 5. Scrape product variants if exists
	productAttributs := make(map[string][]string)
	e.ForEach("label.x-msku__label", func(i int, h *colly.HTMLElement) {
		attribute := h.ChildText("span.x-msku__label-text > span")
		var values []string
		h.ForEach("span.x-msku__select-box-wrapper > select > option:not(:first-child)", func(i int, j *colly.HTMLElement) {
			values = append(values, j.Text)
		})
		productAttributs[attribute] = values
	})

	product.Attributes = productAttributs

	// Step 6. Scrape image links
	var imgLinks []string
	e.ForEach("div.ux-image-carousel-container div[tabindex='0'] div.ux-image-carousel-item.image-treatment.image > img", func(i int, h *colly.HTMLElement) {
		ds := h.Attr("data-src")
		if ds != "" {
			imgLinks = append(imgLinks, ds)
			return
		}
		s := h.Attr("src")
		if s != "" {
			imgLinks = append(imgLinks, s)
			return
		}
	})
	product.ImageLinks = imgLinks

	// Step 7. Scrape product specs
	productSpecs := make(map[string]string)
	e.ForEach("div.vim.x-about-this-item div.ux-layout-section-evo__col", func(i int, h *colly.HTMLElement) {
		label := h.ChildText("div.ux-labels-values__labels-content span.ux-textspans")
		value := h.ChildText("div.ux-labels-values__values-content span.ux-textspans")
		productSpecs[label] = value
	})

	product.Specs = productSpecs

	jsonData, err := json.Marshal(product)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	// Write JSON line to file
	_, err = writer.WriteString(string(jsonData) + "\n")
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
}
