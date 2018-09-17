package gomoss

import (
	"context"
	"errors"
	"fmt"
	"github.com/anaskhan96/soup"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// Region specifies similarity from line a to line b
type Region struct {
	From int64
	To   int64
}

// Source is structure that contains plagiarism result of a source code
type Source struct {
	Filename   string
	Similarity float64
	Regions    []Region
}

// Match compares two source codes potential plagiarism results
type Match struct {
	Left, Right Source
}

// Report is data extracted data from Moss URL
type Report struct {
	Matches []Match
}

var (
	ErrInvalidRegion = errors.New("Invalid region")
)

func mustParse(url *url.URL, rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return u
}

// FindLinks function finds all analysis related hrefs in HTML
func findLinks(ref *url.URL) []string {
	resp, _ := soup.Get(ref.String())

	doc := soup.HTMLParse(resp)
	anchors := doc.FindAll("a")
	exist := make(map[string]bool)
	var links []string
	for _, anchor := range anchors {
		link := anchor.Attrs()["href"]
		if strings.Contains(link, "match") && !exist[link] {
			exist[link] = true
			links = append(links, link)
		}
	}
	return links
}

// DownloadURL downloads HTML to file from given URL
func DownloadURL(ref *url.URL, filename string) error {
	resp, err := soup.Get(ref.String())
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	defer f.Close()
	if err != nil {
		return err
	}

	reg := regexp.MustCompile(`http:\/\/moss.stanford.edu\/results\/\d*\/`)
	resp = reg.ReplaceAllString(resp, "")

	_, err = f.Write([]byte(resp))
	if err != nil {
		return err
	}

	return nil
}

// Download downloads all HTML files from given URL recursively
func Download(ctx context.Context, ref *url.URL, dirName string) error {
	dir, err := os.Stat(dirName)
	if err != nil {
		return err
	}
	mode := dir.Mode()
	if !mode.IsDir() {
		return fmt.Errorf("Dir is not directory")
	}

	links := findLinks(ref)
	DownloadURL(mustParse(ref, "index.html"), path.Join(dirName, "index.html"))
	for index := range links {
		select {
		case <-ctx.Done():
			return nil
		default:
			filename := fmt.Sprintf("match%d.html", index)
			DownloadURL(mustParse(ref, filename), path.Join(dirName, filename))
			ref.Path = fmt.Sprintf("match%d-top.html", index)
			DownloadURL(mustParse(ref, filename), path.Join(dirName, filename))
			ref.Path = fmt.Sprintf("match%d-0.html", index)
			DownloadURL(mustParse(ref, filename), path.Join(dirName, filename))
			ref.Path = fmt.Sprintf("match%d-1.html", index)
			DownloadURL(mustParse(ref, filename), path.Join(dirName, filename))
		}
	}
	return nil
}

func parseRegion(text string) (*Region, error) {
	chunk := strings.Split(text, "-")
	if len(chunk) != 2 {
		return nil, ErrInvalidRegion
	}
	from, err := strconv.ParseInt(chunk[0], 10, 32)
	if err != nil {
		return nil, err
	}
	to, err := strconv.ParseInt(chunk[1], 10, 32)
	if err != nil {
		return nil, err
	}
	return &Region{
		From: from,
		To:   to,
	}, nil
}

func parsePercentage(s string) (float64, error) {
	valueStr := regexp.MustCompile(`[0-9]+%`).FindString(s)
	value, err := strconv.ParseFloat(valueStr[:len(valueStr)-1], 64)
	if err != nil {
		return 0, err
	}
	return value / 100, nil
}

func parseFilename(s string) string {
	value := regexp.MustCompile(`\ \(.*\)`).ReplaceAllString(s, "")
	return value
}

// Extract extracts data from given Moss URL into Report struct
func Extract(ctx context.Context, ref *url.URL) (*Report, error) {
	links := findLinks(ref)
	report := Report{
		Matches: []Match{},
	}
	for index := range links {
		select {
		case <-ctx.Done():
			return nil, nil
		default:
			ref.Path = fmt.Sprintf("match%d-top.html", index)
			res, err := soup.Get(ref.String())
			if err != nil {
				return nil, fmt.Errorf("Unable to get url %s", ref.String())
			}
			doc := soup.HTMLParse(res)
			anchors := doc.FindAll("a")
			headers := doc.FindAll("th")

			leftSimilarity, err := parsePercentage(headers[0].Text())
			if err != nil {
				return nil, err
			}
			rightSimilarity, err := parsePercentage(headers[2].Text())
			if err != nil {
				return nil, err
			}

			match := Match{
				Left: Source{
					Filename:   parseFilename(headers[0].Text()),
					Similarity: leftSimilarity,
					Regions:    []Region{},
				},
				Right: Source{
					Filename:   parseFilename(headers[2].Text()),
					Similarity: rightSimilarity,
					Regions:    []Region{},
				},
			}
			for idx := 0; idx < len(anchors); idx += 4 {
				leftRegion, err := parseRegion(anchors[idx].Text())
				if err != nil {
					return nil, fmt.Errorf("Unable to recognize line number")
				}
				rightRegion, err := parseRegion(anchors[idx+2].Text())
				if err != nil {
					return nil, fmt.Errorf("Unable to line number")
				}

				match.Left.Regions = append(match.Left.Regions, *leftRegion)
				match.Right.Regions = append(match.Right.Regions, *rightRegion)
			}
			report.Matches = append(report.Matches, match)
		}
	}
	return &report, nil
}
