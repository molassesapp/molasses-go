package molasses

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// ClientOptions - for the Molasses client
// Api Key is the required field
// URL can be updated if you are using a hosted version of Molasses
// Debug - whether to log debug info
// HTTPClient - Pass in your own http client
type ClientOptions struct {
	APIKey     string
	URL        string
	Debug      bool
	HTTPClient *http.Client
}

// Client - The client to interface with Molasses
type Client interface {
	IsActive(key string, user ...User) bool
}

type client struct {
	httpClient    *http.Client
	apiKey        string
	url           string
	debug         bool
	etag          string
	initiated     bool
	featuresCache map[string]feature
	refreshTicker *time.Ticker
}

// Init - Creates a new client to interface with Molasses.
// Receives a ClientOptions
func Init(options ClientOptions) (Client, error) {

	molassesClient := &client{
		httpClient:    options.HTTPClient,
		apiKey:        options.APIKey,
		debug:         options.Debug,
		url:           options.URL,
		refreshTicker: time.NewTicker(15 * time.Second),
	}

	if molassesClient.httpClient == nil {
		molassesClient.httpClient = &http.Client{}
	}

	if molassesClient.apiKey == "" {
		return &client{}, errors.New("API KEY must be supplied")
	}

	if molassesClient.httpClient == nil {
		molassesClient.httpClient = &http.Client{}
	}

	if molassesClient.url == "" {
		molassesClient.url = "https://api.molasses.app"
	}

	molassesClient.featuresCache = make(map[string]feature)
	molassesClient.fetchFeatures()
	go molassesClient.refresh()
	return molassesClient, nil
}

func (c *client) refresh() {
	c.fetchFeatures()
	for {
		select {
		case <-c.refreshTicker.C:
			c.fetchFeatures()
		}
	}
}

type features struct {
	Features []feature `json:"features"`
}
type featuresResponse struct {
	Data features `json:"data"`
}

// IsActive - Check to see if a feature is active for a user
// You must pass the key of the feature (ex. SHOW_USER_ONBOARDING) and optionally pass the user who you are evaluating.
// if you pass more than 1 user value, the first will only be evaluated
func (c *client) IsActive(key string, user ...User) bool {
	switch len(user) {
	case 0:
		return isActive(c.featuresCache[key], nil)
	default:
		return isActive(c.featuresCache[key], &user[0])
	}
}

func (c *client) fetchFeatures() error {
	req, err := http.NewRequest("GET", c.url+"/v1/get-features", nil)
	if err != nil {
		return err
	}
	if c.etag != "" {
		req.Header.Add("If-None-Match", c.etag)
	}
	req.Header.Add("Authorization", "Bearer "+c.apiKey)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusNotModified {
		return nil
	}
	var b featuresResponse
	_ = json.NewDecoder(res.Body).Decode(&b)
	for _, feature := range b.Data.Features {
		key := feature.Key
		c.featuresCache[key] = feature
	}
	c.etag = res.Header.Get("Etag")
	return nil
}
