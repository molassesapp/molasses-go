/*
Package molasses is a Go SDK for Molasses. It allows you to evaluate user's status for a feature. It also helps simplify logging events for A/B testing.

Molasses uses polling to check if you have updated features. Once initialized, it takes microseconds to evaluate if a user is active.
*/
package molasses

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// ClientOptions - The options for the Molasses client to start, the APIKey is required
type ClientOptions struct {
	APIKey     string       // APIKey is the required field.
	URL        string       // URL can be updated if you are using a hosted version of Molasses
	Debug      bool         // Debug - whether to log debug info
	HTTPClient *http.Client // HTTPClient - Pass in your own http client
}

type ClientInterface interface {
	IsActive(key string, user ...User) bool
}
type Client struct {
	client
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
func Init(options ClientOptions) (ClientInterface, error) {

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
		molassesClient.url = "https://us-central1-molasses-36bff.cloudfunctions.net/"
	}

	molassesClient.featuresCache = make(map[string]feature)
	molassesClient.fetchFeatures()
	go molassesClient.refresh()
	return molassesClient, nil
}

// IsActive - Check to see if a feature is active for a user.
// You must pass the key of the feature (ex. SHOW_USER_ONBOARDING) and optionally pass the user who you are evaluating.
// if you pass more than 1 user value, the first will only be evaluated
func (c *Client) IsActive(key string, user ...User) bool {
	switch len(user) {
	case 0:
		return isActive(c.featuresCache[key], nil)
	default:
		return isActive(c.featuresCache[key], &user[0])
	}
}
func (c *Client) refresh() {
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

func (c *client) uploadEvent() {

	// private uploadEvent(eventOptions: EventOptions) {
	req, err := http.NewRequest("POST", c.url+"/analytics", nil)
	if err != nil {
		return err
	}
	if c.etag != "" {
		req.Header.Add("If-None-Match", c.etag)
	}
	req.Header.Add("Authorization", "Bearer "+c.apiKey)
	go c.httpClient.Do(req)
	//   const headers = { Authorization: "Bearer " + this.options.APIKey }
	//   const data = {
	//     ...eventOptions,
	//     tags: JSON.stringify(eventOptions.tags),
	//   }
	//   this.axios.post("/analytics", data, {
	//     headers,
	//   })
	// }
}

type eventOptions struct {
	FeatureID   string            `json:"featureId"`
	UserID      string            `json:"userId"`
	FeatureName string            `json:"featureName"`
	Event       string            `json:"event"`
	Tags        map[string]string `json:"tags"`
	TestType    string            `json:"testType"`
}

func (c *Client) fetchFeatures() error {
	req, err := http.NewRequest("GET", c.url+"/get-features", nil)
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
