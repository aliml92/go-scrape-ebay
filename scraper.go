package goscrapeebay

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"
)

var ErrContextTimeout = errors.New("context timeout")

type Config struct {
	OutputFile                string
	TargetURL                 string
	CategoriesFile            string
	MaxRetriesCategories      int
	MaxRetriesProducts        int
	MaxCategoriesPerPage      int
	MaxProductsPerPage        int
	SkipCategoryScraping      bool 
	LogLevel                  string
	CacheDir                  string

	// Delay is the duration to wait before creating a new request to the matching domains
	Delay time.Duration

	// RandomDelay is the extra randomized duration to wait added to Delay before creating a new request
	RandomDelay time.Duration
}

// LeveledLogger interface
type LeveledLogger interface {
	Error(msg string, args ...any)
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
}

// slogAdapter adapts slog.Logger to our LeveledLogger interface
type slogAdapter struct {
	logger *slog.Logger
}

func (s *slogAdapter) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

func (s *slogAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

func (s *slogAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

func (s *slogAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

func createLogger(level string) (LeveledLogger, error) {
	var logLevel slog.Level
	switch level {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log level: %s", level)
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}
	handler := slog.NewJSONHandler(os.Stderr, opts)
	logger := slog.New(handler)

	return &slogAdapter{logger: logger}, nil
}

type EbayScraper struct {
	config Config
	logger LeveledLogger
}

func NewEbayScraper(cfg Config) (*EbayScraper, error) {
	logger, err := createLogger(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("error creating logger: %w", err)
	}

	return &EbayScraper{
		config: cfg,
		logger: logger,
	}, nil
}

func (s *EbayScraper) Run() error {
	if !s.config.SkipCategoryScraping {
		if err := s.runCategoryScrape(); err != nil {
			return err
		}		
	}

	return s.runProductsScrape()
}

func (s *EbayScraper) runCategoryScrape() error {
	rootCategoryURL, err := url.Parse(s.config.TargetURL)
	if err != nil {
		return fmt.Errorf("error parsing URL: %w", err)
	}

	file, err := os.Create(s.config.CategoriesFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	defer w.Flush()

	// Retry loop
	var lastErr error
	for i := 0; i < s.config.MaxRetriesCategories; i++ {
		err = s.scrapeLeafCateories(rootCategoryURL, w)
		if err == nil {
			s.logger.Info("Scraping succeeded")
			break
		}

		if errors.Is(err, ErrContextTimeout) {
			s.logger.Warn("Context timeout, retrying", "attempt", i+1)
		} else {
			s.logger.Error("Scraping error", "error", err)
			break
		}
		lastErr = err
	}

	if lastErr != nil {
		s.logger.Error("Failed to collect categories after maximum attempts",
			"max_attempts", s.config.MaxRetriesCategories,
			"error", lastErr)
	}

	return nil
}

func (s *EbayScraper) runProductsScrape() error {
	s.logger.Info("Started scraping product details pages")

	outputFile, err := os.Create(s.config.OutputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer outputFile.Close()

	w := bufio.NewWriter(outputFile)
	defer w.Flush()

	categoriesFile, err := os.Open(s.config.CategoriesFile)
	if err != nil {
		return fmt.Errorf("error opening urls file: %w", err)
	}
	defer categoriesFile.Close()
	
	var (
		line []byte
		readErr error
	    lastErr error
		pos int64
	)

	lineNum := 1

	for i := 0; i < s.config.MaxRetriesProducts; i++ {
		if _, err := categoriesFile.Seek(pos, io.SeekStart); err != nil {
			return err
		}
	
		r := bufio.NewReader(categoriesFile)

		var errChan chan error
		c := s.setupProductDetailsCollector(w, errChan)

		for {
			errChan = make(chan error, 1)
			line, readErr = r.ReadBytes('\n')
			fmt.Printf("[line:%d pos:%d] %q\n", lineNum, pos, line)

			if readErr != nil {
				break
			}

			pos += int64(len(line))
			lineNum++

			leafURL := strings.Split(string(line), "\n")[0] 

			u, err := url.Parse(leafURL)
			if err != nil {
				s.logger.Error("URL parsing error", "error", err, "category", leafURL)
				continue
			}
			
			go func() {
				err := c.Visit(u.String())
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
			
			err = <-errChan
			if err == nil {
				s.logger.Info("Details page scraped", "url", u.String())
				continue
			}
	
			if errors.Is(err, ErrContextTimeout) {
				s.logger.Warn("Context timeout, retrying", "attempt", i+1)
				break 
			} else {
				s.logger.Error("Failed to scrape product details page", "error", err, "category", leafURL)
			}

			lastErr = err
		}

		if readErr != nil && readErr != io.EOF {
			s.logger.Error("Failed to read file", "err", readErr.Error())
			return err
		}
	}


	if lastErr != nil {
		s.logger.Error("Failed to scraped products after maximum attempts",
			"max_attempts", s.config.MaxRetriesProducts,
			"error", lastErr)
	}

	return nil
}