package gomoss

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"testing"
)

func TestDownloadURL(t *testing.T) {
	handler := http.FileServer(http.Dir("./testdata"))
	ts := httptest.NewServer(handler)
	defer ts.Close()

	path, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	html, err := path.Parse("index.html")
	if err != nil {
		t.Fatal(err)
	}
	err = DownloadURL(html, "a.html")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("a.html")

	// Check downloaded html
	html1, err := ioutil.ReadFile("a.html")
	if err != nil {
		t.Fatal(err)
	}
	html2, err := ioutil.ReadFile("testdata/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(html1, html2) {
		t.Errorf("%v\n", "Downloaded html and server html are not equal")
	}
}

func TestDownloadMoss(t *testing.T) {
	handler := http.FileServer(http.Dir("./testdata"))
	ts := httptest.NewServer(handler)
	defer ts.Close()

	urlPath, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	dir := "moss.education.com"
	err = os.Mkdir(dir, 644)
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	err = Download(ctx, urlPath, dir)
	if err != nil {
		t.Fatal(err)
	}

	// Check downloaded html
	err = filepath.Walk(dir, func(urlPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		html1, err := ioutil.ReadFile(path.Join(dir, filepath.Base(urlPath)))
		if err != nil {
			return err
		}
		html2, err := ioutil.ReadFile(path.Join("testdata", filepath.Base(urlPath)))
		if err != nil {
			return err
		}
		if !bytes.Equal(html1, html2) {
			return errors.New(filepath.Base(urlPath) + " are not equal")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExtractMoss(t *testing.T) {
	handler := http.FileServer(http.Dir("./testdata"))
	ts := httptest.NewServer(handler)
	defer ts.Close()

	urlPath, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	report, err := Extract(ctx, urlPath)
	if err != nil {
		t.Fatalf("Unable to extract report")
	}

	match := report.Matches[0]

	if match.Left.Filename != "gomoss/codes/aaa.cpp" || match.Right.Filename != "gomoss/codes/bbb.cpp" {
		t.Errorf("Gomoss extract moss filenames are incorrect")
	}
	if match.Left.Similarity != 0.82 || match.Right.Similarity != 0.79 {
		t.Errorf("Gomoss extract moss percentages are incorrect")
	}
}

func ExampleDownload() {
	urlStruct, err := url.Parse("http://moss.stanford.edu/results/NUMBER")
	if err != nil {
		panic(err)
	}

	// Context is used to terminate Download process
	ctx := context.Background()
	dir := "moss.education.com"
	err = os.Mkdir(dir, 644)
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	err = Download(ctx, urlStruct, dir)
	if err != nil {
		panic(err)
	}
}

func ExampleExtract() {
	// Context is used to terminate Extract process
	urlStruct, err := url.Parse("http://moss.stanford.edu/results/NUMBER")
	if err != nil {
		panic(err)
	}

	// Context is used to terminate Download process
	ctx := context.Background()
	results, err := Extract(ctx, urlStruct)
	if err != nil {
		panic(err)
	}

	// View results in json
	for _, match := range results.Matches {
		js, _ := json.Marshal(match)
		fmt.Println(string(js))
	}
}
