package torznab

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"time"

	"github.com/hexsans/hexmagnet/internal/model"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
)

// Namespaces used by the Torznab feed format.
const (
	serverName = "HexMagnet"
	atomNS     = "http://www.w3.org/2005/Atom"
	torznabNS  = "http://torznab.com/schemas/2015/feed"
)

// Newznab error codes.
const (
	attrYes                 = "yes"
	errCodeParameterMissing = 100
	errCodeParameterInvalid = 101
	errCodeNoSuchFunction   = 200
	errCodeNoSuchItem       = 300
	errCodeServerError      = 500
	errCodeUnauthorized     = 100
)

type rss struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Atom    string     `xml:"xmlns:atom,attr"`
	Torznab string     `xml:"xmlns:torznab,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Link        string    `xml:"link"`
	Language    string    `xml:"language"`
	Generator   string    `xml:"generator"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string    `xml:"title"`
	GUID        rssGUID   `xml:"guid"`
	PubDate     string    `xml:"pubDate"`
	Description string    `xml:"description"`
	Link        string    `xml:"link"`
	Size        int64     `xml:"size"`
	Category    string    `xml:"category"`
	Enclosure   enclosure `xml:"enclosure"`
	Attrs       []rssAttr `xml:"torznab:attr"`
}

type rssGUID struct {
	Value       string `xml:",chardata"`
	IsPermaLink bool   `xml:"isPermaLink,attr"`
}

type enclosure struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Length int64  `xml:"length,attr"`
}

type rssAttr struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// searchResult is the data needed to render a Torznab item.
type searchResult struct {
	InfoHash      string
	Name          string
	Size          int64
	Seeders       *int32
	Leechers      *int32
	ContentType   *string
	ContentSource *string
	ContentID     *string
	CreatedAt     time.Time
}

func newSearchResult(row dbsearch.TorrentSearchRow) searchResult {
	return searchResult{
		InfoHash:      row.InfoHash,
		Name:          row.TorrentName,
		Size:          row.Size,
		Seeders:       row.Seeders,
		Leechers:      row.Leechers,
		ContentType:   row.ContentType,
		ContentSource: row.ContentSource,
		ContentID:     row.ContentID,
		CreatedAt:     row.CreatedAt,
	}
}

// feedFor builds a Torznab RSS document from search results.
func feedFor(baseURL *url.URL, results []dbsearch.TorrentSearchRow) ([]byte, error) {
	items := make([]rssItem, 0, len(results))

	for _, row := range results {
		item, err := itemFor(baseURL, newSearchResult(row))
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	channel := rssChannel{
		Title:       serverName,
		Description: "HexMagnet torrent indexer (Torznab)",
		Link:        baseURL.String(),
		Language:    "en-us",
		Generator:   serverName,
		Items:       items,
	}

	return xml.MarshalIndent(rss{
		Version: "2.0",
		Atom:    atomNS,
		Torznab: torznabNS,
		Channel: channel,
	}, "", "  ")
}

func itemFor(baseURL *url.URL, r searchResult) (rssItem, error) {
	downloadURL, err := downloadURLFor(baseURL, r.InfoHash)
	if err != nil {
		return rssItem{}, err
	}

	seeders := int32(0)
	if r.Seeders != nil {
		seeders = *r.Seeders
	}

	leechers := int32(0)
	if r.Leechers != nil {
		leechers = *r.Leechers
	}

	category := catOther

	var categoryName string

	if r.ContentType != nil {
		if ct, err := model.ParseContentType(*r.ContentType); err == nil {
			category = categoryForContentType(ct)
		}
	}

	categoryName = newznabCategoryName(category)

	attrs := []rssAttr{
		{Name: "infohash", Value: r.InfoHash},
		{Name: "magneturl", Value: magnetURI(r)},
		{Name: "seeders", Value: fmt.Sprintf("%d", seeders)},
		{Name: "leechers", Value: fmt.Sprintf("%d", leechers)},
		{Name: "peers", Value: fmt.Sprintf("%d", seeders+leechers)},
		{Name: "size", Value: fmt.Sprintf("%d", r.Size)},
		{Name: "category", Value: fmt.Sprintf("%d", category)},
	}

	if r.ContentSource != nil && r.ContentID != nil {
		attrs = append(attrs, rssAttr{
			Name:  contentAttrName(*r.ContentSource),
			Value: *r.ContentID,
		})
	}

	return rssItem{
		Title:       r.Name,
		GUID:        rssGUID{Value: r.InfoHash, IsPermaLink: false},
		PubDate:     r.CreatedAt.UTC().Format(time.RFC1123Z),
		Description: r.Name,
		Link:        magnetURI(r),
		Size:        r.Size,
		Category:    categoryName,
		Enclosure: enclosure{
			URL:    downloadURL,
			Type:   "application/x-bittorrent",
			Length: 0,
		},
		Attrs: attrs,
	}, nil
}

// magnetURI builds a magnet link for a torrent result.
func magnetURI(r searchResult) string {
	params := url.Values{}
	params.Set("xt", "urn:btih:"+r.InfoHash)
	params.Set("dn", r.Name)

	if r.Size > 0 {
		params.Set("xl", fmt.Sprintf("%d", r.Size))
	}

	return "magnet:?" + params.Encode()
}

func contentAttrName(source string) string {
	switch source {
	case model.SourceImdb:
		return "imdbid"
	case model.SourceTmdb:
		return "tmdbid"
	case model.SourceTvdb:
		return "tvdbid"
	default:
		return source
	}
}

// downloadURLFor builds the URL of the .torrent download endpoint.
func downloadURLFor(baseURL *url.URL, infoHash string) (string, error) {
	if infoHash == "" {
		return "", fmt.Errorf("empty info hash")
	}

	u := *baseURL
	u.Path = "/api/torrents/" + infoHash + "/download"

	return u.String(), nil
}

func newznabCategoryName(cat int) string {
	switch cat {
	case catMovies:
		return "Movies"
	case catTV:
		return "TV"
	case catAudio:
		return "Audio"
	case catBooks:
		return "Books"
	case catBooksComics:
		return "Books > Comics"
	case catBooksAudiobook:
		return "Books > Audiobook"
	case catConsole:
		return "Console"
	case catSoftware:
		return "PC"
	case catXXX:
		return "XXX"
	default:
		return "Other"
	}
}

type caps struct {
	XMLName      xml.Name     `xml:"caps"`
	Server       capsServer   `xml:"server"`
	Limits       capsLimits   `xml:"limits"`
	Registration capsReg      `xml:"registration"`
	Searching    capsSearch   `xml:"searching"`
	Categories   capsCategory `xml:"categories"`
	Genres       capsGenres   `xml:"genres"`
}

type capsServer struct {
	Title         string `xml:"title,attr"`
	Version       string `xml:"version,attr"`
	ServerVersion string `xml:"serverversion,attr"`
}

type capsLimits struct {
	Max     int `xml:"max,attr"`
	Default int `xml:"default,attr"`
}

type capsReg struct {
	Available string `xml:"available,attr"`
	Open      string `xml:"open,attr"`
}

type capsSearch struct {
	Search      capsMethod `xml:"search"`
	TvSearch    capsMethod `xml:"tv-search"`
	MovieSearch capsMethod `xml:"movie-search"`
	AudioSearch capsMethod `xml:"audio-search"`
	BookSearch  capsMethod `xml:"book-search"`
}

type capsMethod struct {
	Available string `xml:"available,attr"`
	Supported string `xml:"supportedParams,attr"`
}

type capsCategory struct {
	Categories []capsCat `xml:"category"`
}

type capsCat struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

type capsGenres struct{}

func capabilitiesFor(cfg Config) []byte {
	var cats []capsCat

	for _, cat := range baseCategories {
		if !allowedCategories(cfg.Categories, cat) {
			continue
		}

		cats = append(cats, capsCat{
			ID:   fmt.Sprintf("%d", cat),
			Name: newznabCategoryName(cat),
		})
	}

	doc := caps{
		Server: capsServer{
			Title:         serverName,
			Version:       "1.0",
			ServerVersion: "1.0",
		},
		Limits: capsLimits{
			Max:     capResults(0, cfg.MaxResults),
			Default: capResults(0, cfg.MaxResults),
		},
		Registration: capsReg{
			Available: "no",
			Open:      "no",
		},
		Searching: capsSearch{
			Search: capsMethod{
				Available: attrYes,
				Supported: "q,cat,apikey,limit,offset",
			},
			TvSearch: capsMethod{
				Available: attrYes,
				Supported: "q,season,ep,imdbid,tmdbid,tvdbid,cat,apikey,limit,offset",
			},
			MovieSearch: capsMethod{
				Available: attrYes,
				Supported: "q,imdbid,tmdbid,cat,apikey,limit,offset",
			},
			AudioSearch: capsMethod{
				Available: attrYes,
				Supported: "q,artist,album,label,cat,apikey,limit,offset",
			},
			BookSearch: capsMethod{
				Available: attrYes,
				Supported: "q,title,author,cat,apikey,limit,offset",
			},
		},
		Categories: capsCategory{Categories: cats},
		Genres:     capsGenres{},
	}

	out, _ := xml.MarshalIndent(doc, "", "  ")

	return out
}

// errorResponse builds a Newznab-style error document.
func errorResponse(code int, description string) []byte {
	doc := struct {
		XMLName     xml.Name `xml:"error"`
		Code        int      `xml:"code,attr"`
		Description string   `xml:"description,attr"`
	}{
		Code:        code,
		Description: description,
	}

	out, _ := xml.MarshalIndent(doc, "", "  ")

	return out
}
