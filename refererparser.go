package refererparser

import (
	"io/ioutil"
	"net/url"
	"strings"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"
)

type refererMapping map[string]struct {
	Source string   `yaml:"source"`
	Medium string   `yaml:"medium"`
	Query  []string `yaml:"query"`
}

func (r refererMapping) lookupByDomain(domain string, currentDomain string, queries url.Values) *ParserDetails {
	referResult := &ParserDetails{
		Medium: "unknown",
		Source: "unknown",
		Term:   "",
	}

	if currentDomain == domain {
		referResult.Medium = "internal"
		referResult.Source = "internal"
		referResult.Known = true
		return referResult
	}

	refer, exist := r[domain]

	if !exist {
		return referResult
	}

	referResult.Medium = refer.Medium
	referResult.Source = refer.Source
	referResult.Known = true
	for _, q := range refer.Query {
		term := queries.Get(q)
		if term != "" {
			referResult.Term = term
			break
		}
	}

	return referResult
}

// New ...
func New(path string) (*Reader, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var refererMap refererMapping
	err = yaml.Unmarshal(data, &refererMap)

	if err != nil {
		return nil, err
	}

	return &Reader{
		refererMap: refererMap,
	}, nil
}

// Reader ...
type Reader struct {
	refererMap refererMapping
}

// ParserDetails ...
type ParserDetails struct {
	Medium string
	Source string
	Term   string
	Known  bool
}

// Parser ...
func (reader *Reader) Parse(url string, currentUrl string) (*ParserDetails, error) {
	if len(url) == 0 || len(currentUrl) == 0 {
		return nil, xerrors.New("empty parameters")
	}

	return reader.LookupReferer(url, currentUrl)
}

func getDomainAlias(puri *url.URL) string {
	splits := strings.SplitAfterN(puri.Host, ".", 2)
	if len(splits) == 2 {
		return splits[1]
	}
	return puri.Host
}

func getPathAlias(puri *url.URL) string {
	splits := strings.Split(puri.Path, "/")
	if len(splits) == 2 {
		return splits[1]
	}
	return ""
}

func (reader *Reader) LookupReferer(uri string, currentUrl string) (*ParserDetails, error) {
	puri, perr := url.Parse(uri)
	if perr != nil {
		return nil, perr
	}

	curi, cerr := url.Parse(currentUrl)
	if cerr != nil {
		return nil, cerr
	}

	domainAlias := getDomainAlias(puri)
	domain := puri.Host
	query := puri.Query()
	path := puri.Path
	pathAlias := getPathAlias(puri)
	currentDomain := curi.Host

	domainQueries := []string{
		domain + path,
		domainAlias + path,
		domain + "/" + pathAlias,
		domainAlias + "/" + pathAlias,
		domain,
		domainAlias,
	}

	var referer *ParserDetails
	for _, domainQ := range domainQueries {
		referer = reader.refererMap.lookupByDomain(domainQ, currentDomain, query)
		if referer.Known {
			break
		}
	}

	return referer, nil

}
