package discogs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	discogsAPI = "https://api.discogs.com"
)

// Options is a set of options to use discogs API client
type Options struct {
	// Discogs API endpoint (optional).
	URL string
	// Currency to use (optional, default is USD).
	Currency string
	// UserAgent to to call discogs api with.
	UserAgent string
	// Token provided by discogs (optional).
	Token string
}

// Discogs is an interface for making Discogs API requests.
type Discogs interface {
	CollectionService
	DatabaseService
	MarketPlaceService
	SearchService
}

type discogs struct {
	CollectionService
	DatabaseService
	SearchService
	MarketPlaceService
}

var header *http.Header

// New returns a new discogs API client.
func New(o *Options) (Discogs, error) {
	header = &http.Header{}

	if o == nil || o.UserAgent == "" {
		return nil, ErrUserAgentInvalid
	}

	header.Add("User-Agent", o.UserAgent)

	cur, err := currency(o.Currency)
	if err != nil {
		return nil, err
	}

	// set token, it's required for some queries like search
	if o.Token != "" {
		header.Add("Authorization", "Discogs token="+o.Token)
	}

	if o.URL == "" {
		o.URL = discogsAPI
	}

	return discogs{
		newCollectionService(o.URL + "/users"),
		newDatabaseService(o.URL, cur),
		newSearchService(o.URL + "/database/search"),
		newMarketPlaceService(o.URL+"/marketplace", cur),
	}, nil
}

// currency validates currency for marketplace data.
// Defaults to the authenticated users currency. Must be one of the following:
// USD GBP EUR CAD AUD JPY CHF MXN BRL NZD SEK ZAR
func currency(c string) (string, error) {
	switch c {
	case "USD", "GBP", "EUR", "CAD", "AUD", "JPY", "CHF", "MXN", "BRL", "NZD", "SEK", "ZAR":
		return c, nil
	case "":
		return "USD", nil
	default:
		return "", ErrCurrencyNotSupported
	}
}

func request(path string, params url.Values, resp interface{}) error {
	return requestWithMethod("GET", path, params, resp)
}

func requestWithMethod(method string, path string, params url.Values, resp interface{}) error {
	return verboseRequest(method, path, params, nil, resp)
}

func requestWithJSONBody(method string, path string, params url.Values, body interface{}, resp interface{}) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return verboseRequest(method, path, params, bytes.NewBuffer(bodyBytes), resp)
}

func verboseRequest(method string, path string, params url.Values, requestBody io.Reader, resp interface{}) error {
	r, err := http.NewRequest(method, path+"?"+params.Encode(), requestBody)
	if err != nil {
		return err
	}
	r.Header = *header
	r.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(r)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		switch response.StatusCode {
		case http.StatusNoContent:
			return nil
		case http.StatusUnauthorized:
			return ErrUnauthorized
		case http.StatusTooManyRequests:
			return ErrTooManyRequests
		default:
			return fmt.Errorf("unknown error: %s", response.Status)
		}
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, &resp)
}
