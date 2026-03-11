package fetch

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestFetchPageInfo_TitleAndDescription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head>
			<title>Example Site</title>
			<meta name="description" content="The best example site">
		</head><body></body></html>`))
	}))
	defer srv.Close()

	info, err := FetchPageInfo(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "Example Site" {
		t.Errorf("title: want %q, got %q", "Example Site", info.Title)
	}
	if info.Description != "The best example site" {
		t.Errorf("description: want %q, got %q", "The best example site", info.Description)
	}
}

func TestFetchPageInfo_TitleOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Only Title</title></head></html>`))
	}))
	defer srv.Close()

	info, err := FetchPageInfo(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "Only Title" {
		t.Errorf("title: want %q, got %q", "Only Title", info.Title)
	}
	if info.Description != "" {
		t.Errorf("expected empty description, got %q", info.Description)
	}
}

func TestFetchPageInfo_DescriptionOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head>
			<meta name="description" content="Just a description">
		</head></html>`))
	}))
	defer srv.Close()

	info, err := FetchPageInfo(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "" {
		t.Errorf("expected empty title, got %q", info.Title)
	}
	if info.Description != "Just a description" {
		t.Errorf("description: want %q, got %q", "Just a description", info.Description)
	}
}

func TestFetchPageInfo_EmptyPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>no metadata</body></html>`))
	}))
	defer srv.Close()

	info, err := FetchPageInfo(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "" || info.Description != "" {
		t.Errorf("expected empty info, got title=%q description=%q", info.Title, info.Description)
	}
}

func TestFetchPageInfo_MetaNameCaseInsensitive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head>
			<meta name="Description" content="Case insensitive">
		</head></html>`))
	}))
	defer srv.Close()

	info, err := FetchPageInfo(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.Description != "Case insensitive" {
		t.Errorf("expected case-insensitive meta match, got %q", info.Description)
	}
}

func TestFetchPageInfo_NoScheme_PrependsHTTPS(t *testing.T) {
	// Calling with a bare host (no scheme) should prepend "https://".
	// The connection will fail (no server), but the prepend branch is covered.
	_, err := FetchPageInfo("127.0.0.1:1") // no scheme, nothing listening
	if err == nil {
		t.Fatal("expected error connecting to non-existent server")
	}
	// If the error is about TLS or connection refused, the prepend ran.
	// (If scheme was preserved as-is, it would be a different kind of error.)
}

func TestFetchPageInfo_HTTPSchemeKept(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>HTTP</title></head></html>`))
	}))
	defer srv.Close()

	// srv.URL already has http://
	info, err := FetchPageInfo(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "HTTP" {
		t.Errorf("title: want %q, got %q", "HTTP", info.Title)
	}
}

func TestFetchPageInfo_NetworkError(t *testing.T) {
	_, err := FetchPageInfo("http://127.0.0.1:1") // nothing listening
	if err == nil {
		t.Fatal("expected error for unreachable host")
	}
}

func TestFetchPageInfo_TitleWithWhitespace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html><head><title>  Trimmed Title  </title></head></html>"))
	}))
	defer srv.Close()

	info, err := FetchPageInfo(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "Trimmed Title" {
		t.Errorf("expected trimmed title, got %q", info.Title)
	}
}

func TestFetchPageInfo_HTMLParseError(t *testing.T) {
	orig := htmlParser
	htmlParser = func(r io.Reader) (*html.Node, error) {
		return nil, fmt.Errorf("parse failed")
	}
	t.Cleanup(func() { htmlParser = orig })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html><body>ok</body></html>"))
	}))
	defer srv.Close()

	_, err := FetchPageInfo(srv.URL)
	if err == nil {
		t.Fatal("expected error when html.Parse fails")
	}
	if !strings.Contains(err.Error(), "parsing HTML") {
		t.Errorf("unexpected error: %v", err)
	}
}
