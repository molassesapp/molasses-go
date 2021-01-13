/*
Package molasses is a Go SDK for Molasses. It allows you to evaluate user's status for a feature. It also helps simplify logging events for A/B testing.

Molasses uses polling to check if you have updated features. Once initialized, it takes microseconds to evaluate if a user is active.
*/
package molasses

import (
	"bytes"
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
	SendEvents *bool
}

func Bool(value bool) *bool {
	return &value
}

type ClientInterface interface {
	IsActive(key string, user ...User) bool
	Stop()
	IsInitiated() bool
	ExperimentSuccess(key string, user User, additionalDetails map[string]string)
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
	sendEvents    bool
}

// Init - Creates a new client to interface with Molasses.
// Receives a ClientOptions
func Init(options ClientOptions) (ClientInterface, error) {
	var sendEvents bool = true
	if options.SendEvents != nil {
		sendEvents = *options.SendEvents
	}

	molassesClient := &client{
		httpClient:    options.HTTPClient,
		apiKey:        options.APIKey,
		debug:         options.Debug,
		url:           options.URL,
		refreshTicker: time.NewTicker(15 * time.Second),
		sendEvents:    sendEvents,
	}

	if molassesClient.httpClient == nil {
		molassesClient.httpClient = &http.Client{}
	}

	if molassesClient.apiKey == "" {
		return &client{}, errors.New("API KEY must be supplied")
	}

	if molassesClient.url == "" {
		molassesClient.url = "https://sdk.molasses.app/v1"
	}

	molassesClient.featuresCache = make(map[string]feature)
	molassesClient.fetchFeatures()
	go molassesClient.refresh()
	return molassesClient, nil
}

// IsActive - Check to see if a feature is active for a user.
// You must pass the key of the feature (ex. SHOW_USER_ONBOARDING) and optionally pass the user who you are evaluating.
// if you pass more than 1 user value, the first will only be evaluated
func (c *client) IsActive(key string, user ...User) bool {
	f := c.featuresCache[key]
	switch len(user) {
	case 0:
		return isActive(f, nil)
	default:
		result := isActive(f, &user[0])
		var r = "experiment"
		if result {
			r = "control"
		}
		defer c.uploadEvent(eventOptions{
			Event:       "experiment_started",
			Tags:        user[0].Params,
			UserID:      user[0].ID,
			FeatureID:   f.ID,
			FeatureName: key,
			TestType:    r,
		})
		return result
	}
}

func (c *client) IsInitiated() bool {
	return c.initiated
}

func (c *client) ExperimentSuccess(key string, user User, additionalDetails map[string]string) {

	if !c.initiated {
		return
	}

	f := c.featuresCache[key]
	result := isActive(f, &user)

	var r = "experiment"
	if result {
		r = "control"
	}

	for k, v := range additionalDetails {
		user.Params[k] = v
	}

	c.uploadEvent(eventOptions{
		Event:       "experiment_success",
		Tags:        user.Params,
		UserID:      user.ID,
		FeatureID:   f.ID,
		FeatureName: key,
		TestType:    r,
	})
}

func (c *client) Stop() {
	c.refreshTicker.Stop()
	c.initiated = false
}

func (c *client) refresh() {
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

func (c *client) uploadEvent(e eventOptions) error {
	if !c.sendEvents {
		return nil
	}
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.url+"/analytics", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.etag != "" {
		req.Header.Add("If-None-Match", c.etag)
	}
	req.Header.Add("Authorization", "Bearer "+c.apiKey)
	go c.httpClient.Do(req)
	return nil
}

type eventOptions struct {
	FeatureID   string            `json:"featureId"`
	UserID      string            `json:"userId"`
	FeatureName string            `json:"featureName"`
	Event       string            `json:"event"`
	Tags        map[string]string `json:"tags"`
	TestType    string            `json:"testType"`
}

func (c *client) fetchFeatures() error {
	req, err := http.NewRequest("GET", c.url+"/features", nil)
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
	c.initiated = true
	c.etag = res.Header.Get("Etag")
	return nil
}
