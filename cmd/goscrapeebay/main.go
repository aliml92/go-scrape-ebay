package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	gse "github.com/aliml92/go-scrape-ebay"
)

func main() {
	var cfg gse.Config
	var showHelp bool

	flag.StringVar(&cfg.OutputFile, "output", "output/scraped_data.jsonl", "Output file path")
	flag.StringVar(&cfg.TargetURL, "url", "", "Target URL to scrape (required)")
	flag.StringVar(&cfg.CategoriesFile, "categories-file", "leaf_categories.txt", "File to store leaf category URLs")
	flag.IntVar(&cfg.MaxRetriesCategories, "retries-categories", 3, "Maximum number of retry attempts for category scraping")
	flag.IntVar(&cfg.MaxRetriesProducts, "retries-products", 3, "Maximum number of retry attempts for product scraping")
	flag.IntVar(&cfg.MaxCategoriesPerPage, "max-categories", 5, "Maximum number of child categories to scrape per page")
	flag.IntVar(&cfg.MaxProductsPerPage, "max-products", 20, "Maximum number of products to scrape per page")
	flag.BoolVar(&cfg.SkipCategoryScraping, "skip-category-scraping", false, "skip category scraping")
	flag.StringVar(&cfg.LogLevel, "log-level", "DEBUG", "Log level (DEBUG, INFO, WARN, ERROR)")
	flag.StringVar(&cfg.CacheDir, "cache-dir", "./cache", "Directory for caching scraped data")
	flag.DurationVar(&cfg.Delay, "delay", 2*time.Second, "Delay between requests")
	flag.DurationVar(&cfg.RandomDelay, "random-delay", 1*time.Second, "Maximum random delay added to the base delay")
	flag.BoolVar(&showHelp, "help", false, "Show help message")

	flag.Usage = customUsage
	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if cfg.TargetURL == "" {
		fmt.Fprintln(os.Stderr, "Error: -url flag is required")
		flag.Usage()
		os.Exit(1)
	}

	scraper, err := gse.NewEbayScraper(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating scraper: %v\n", err)
		os.Exit(1)
	}

	if err := scraper.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func customUsage() {
	fmt.Printf("Usage: %s -url=TARGET_URL [OPTIONS]\n\n", os.Args[0])
	fmt.Println("A scraper for eBay products data.")
	fmt.Println("\nOptions:")
	flag.PrintDefaults()
	fmt.Println("\nExample:")
	fmt.Printf("  %s -url=https://www.ebay.com/b/Toys-Hobbies/220/bn_1865497 -output=custom_output.jsonl -retries-categories=3 -log-level=INFO\n", os.Args[0])
}