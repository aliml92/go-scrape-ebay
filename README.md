# Go Scrape eBay

Go Scrape eBay is a command-line tool written in Go for scraping product data from eBay.

## Installation

To install Go Scrape eBay, make sure you have Go installed on your system, then run:

```bash
go install github.com/aliml92/go-scrape-ebay/cmd/goscrapeebay@latest
```

## Usage

After installation, you can run the tool from anywhere using:
```bash
goscrapeebay [OPTIONS]
```

## Options
- `url`: Target URL to scrape (required)
- `output`: Output file path (default: "output/scraped_data.jsonl")
- `categories-file`: File to store leaf category URLs (default: "leaf_categories.txt")
- `retries-categories`: Maximum number of retry attempts for category scraping (default: 3)
- `retries-products`: Maximum number of retry attempts for product scraping (default: 3)
- `max-categories`: Maximum number of child categories to scrape per page (default: 5)
- `max-products`: Maximum number of products to scrape per page (default: 20)
- `skip-category-scraping`: Skip category scraping (default: false)
- `log-level`: Log level (DEBUG, INFO, WARN, ERROR) (default: "DEBUG")
- `cache-dir`: Directory for caching scraped data (default: "./cache")
- `delay`: Delay between requests (default: 2s)
- `random-delay`: Maximum random delay added to the base delay (default: 1s)
- `help`: Show help message

## Example
```bash
goscrapeebay -output=custom_output.jsonl -url=https://www.example.com -retries-categories=5 -log-level=DEBUG
```

## Disclaimer
This tool is for educational purposes only. Be sure to comply with eBay's terms of service and robots.txt file when using this scraper.