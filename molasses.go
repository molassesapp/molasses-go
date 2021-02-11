/*
Package molasses is a Go SDK for Molasses. It allows you to evaluate user's status for a feature. It also helps simplify logging events for A/B testing.

Molasses uses polling to check if you have updated features. Once initialized, it takes microseconds to evaluate if a user is active.
*/
package molasses

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	sse "github.com/r3labs/sse/v2"
	"gopkg.in/cenkalti/backoff.v1"
)

// ClientOptions - The options for the Molasses client to start, the APIKey is required
type ClientOptions struct {
	APIKey         string     // APIKey is the required field.
	URL            string     // URL can be updated if you are using a hosted version of Molasses
	Debug          bool       // Debug - whether to log debug info
	HTTPClient     HttpClient // HTTPClient - Pass in your own http client
	AutoSendEvents bool
	Polling        bool
}

type ClientInterface interface {
	IsActive(key string, user ...User) bool
	Stop()
	IsInitiated() bool
	Track(eventName string, user User, additionalDetails map[string]interface{})
	ExperimentStarted(key string, user User, additionalDetails map[string]interface{})
	ExperimentSuccess(key string, user User, additionalDetails map[string]interface{})
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	client
}
type client struct {
	httpClient        HttpClient
	apiKey            string
	url               string
	debug             bool
	etag              string
	polling           bool
	initiated         bool
	isStreamConnected bool
	featuresCache     map[string]feature
	logger            *log.Logger
	sseClient         *sse.Client
	eventsChannel     chan *sse.Event
	refreshTicker     *time.Ticker
	autoSendEvents    bool
}

// Init - Creates a new client to interface with Molasses.
// Receives a ClientOptions
func Init(options ClientOptions) (ClientInterface, error) {
	polling := options.Polling

	baseURL := "https://sdk.molasses.app/v1"
	if options.URL != "" {
		baseURL = options.URL
	}

	molassesLog := log.New(os.Stderr, "[Molasses]", log.LstdFlags)
	sseClient := sse.NewClient(baseURL + "/event-stream")
	sseClient.ResponseValidator = func(c *sse.Client, resp *http.Response) error {
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			// molassesLog.Println("Molasses is unauthorized")
			return errors.New("Molasses is Unauthorized")
		}

		if resp.StatusCode >= 500 {
			// molassesLog.Println(fmt.Sprintf("There is an issue connecting to Molasses status code - %v", resp.StatusCode))
			return fmt.Errorf("There is an issue connecting to Molasses status code - %v", resp.StatusCode)
		}
		return nil
	}

	sseClient.ReconnectNotify = func(err error, backoff time.Duration) {
		molassesLog.Println("Reconnect", err, backoff)
	}
	backoffStrategy := backoff.NewExponentialBackOff()

	backoffStrategy.MaxElapsedTime = 0
	sseClient.ReconnectStrategy = backoffStrategy
	eventsChannel := make(chan *sse.Event)

	molassesClient := &client{
		httpClient:        options.HTTPClient,
		apiKey:            options.APIKey,
		debug:             options.Debug,
		url:               baseURL,
		polling:           polling,
		sseClient:         sseClient,
		logger:            molassesLog,
		isStreamConnected: false,
		eventsChannel:     eventsChannel,
		refreshTicker:     time.NewTicker(15 * time.Second),
		autoSendEvents:    options.AutoSendEvents,
	}

	if molassesClient.httpClient == nil {
		molassesClient.httpClient = &http.Client{}
	}

	if molassesClient.apiKey == "" {
		return &client{}, errors.New("API KEY must be supplied")
	}
	molassesClient.featuresCache = make(map[string]feature)
	if polling {
		if err := molassesClient.fetchFeatures(); err != nil {
			molassesClient.logger.Printf("Error fetching molasses client features %v", err)
		} else {
			molassesClient.logger.Println("Molasses is connected, polling, and initiated")
		}
	} else {
		molassesClient.sseClient.Headers["Authorization"] = "Bearer " + molassesClient.apiKey
		err := sseClient.SubscribeChan("messages", molassesClient.eventsChannel)
		if err != nil {
			return &client{}, errors.New("Failed to connect to Molasses channel")
		}
		sseClient.OnDisconnect(func(c *sse.Client) {
			molassesClient.logger.Printf("Client disconnected")
			molassesClient.isStreamConnected = false
		})
	}

	go molassesClient.refresh()
	return molassesClient, nil
}

// IsActive - Check to see if a feature is active for a user.
// You must pass the key of the feature (ex. SHOW_USER_ONBOARDING) and optionally pass the user who you are evaluating.
// if you pass more than 1 user value, the first will only be evaluated
func (c *client) IsActive(key string, user ...User) bool {
	f, ok := c.featuresCache[key]
	if !ok {
		c.logger.Printf("Warning - feature flag %s not set in environment -", key)
		return false
	}
	switch len(user) {
	case 0:
		return isActive(f, nil)
	default:
		result := isActive(f, &user[0])
		var r = "experiment"
		if result {
			r = "control"
		}
		defer func() {
			if c.autoSendEvents {
				if err := c.uploadEvent(eventOptions{
					Event:       "experiment_started",
					Tags:        user[0].Params,
					UserID:      user[0].ID,
					FeatureID:   f.ID,
					FeatureName: key,
					TestType:    r,
				}); err != nil {
					c.logger.Printf("Error uploading experiment started event- %s", err.Error())
				}
			}

		}()
		return result
	}
}

func (c *client) IsInitiated() bool {
	return c.initiated
}

func (c *client) ExperimentStarted(key string, user User, additionalDetails map[string]interface{}) {

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

	if err := c.uploadEvent(eventOptions{
		Event:       "experiment_started",
		Tags:        user.Params,
		UserID:      user.ID,
		FeatureID:   f.ID,
		FeatureName: key,
		TestType:    r,
	}); err != nil {
		c.logger.Printf("Error uploading event- %s", err.Error())
	}
}

func (c *client) Track(eventName string, user User, additionalDetails map[string]interface{}) {

	for k, v := range additionalDetails {
		user.Params[k] = v
	}

	if err := c.uploadEvent(eventOptions{
		Event:  eventName,
		Tags:   user.Params,
		UserID: user.ID,
	}); err != nil {
		c.logger.Printf("Error uploading event- %s", err.Error())
	}
}

func (c *client) ExperimentSuccess(key string, user User, additionalDetails map[string]interface{}) {

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

	if err := c.uploadEvent(eventOptions{
		Event:       "experiment_success",
		Tags:        user.Params,
		UserID:      user.ID,
		FeatureID:   f.ID,
		FeatureName: key,
		TestType:    r,
	}); err != nil {
		c.logger.Printf("Error uploading event- %s", err.Error())
	}
}

func (c *client) Stop() {
	c.sseClient.Unsubscribe(c.eventsChannel)
	c.refreshTicker.Stop()
	c.initiated = false
}

func (c *client) refresh() {
	for {
		select {
		case res := <-c.eventsChannel:
			data := res.Data
			var f featuresResponse
			err := json.Unmarshal(data, &f)
			if err != nil {
				c.logger.Printf("Error refreshing features - %s", err.Error())
			}
			for _, feature := range f.Data.Features {
				key := feature.Key
				c.featuresCache[key] = feature
			}

			if !c.isStreamConnected {
				c.logger.Println("Molasses is connected")
			}
			if !c.initiated {
				c.logger.Println("Molasses is initiated")
			}
			c.isStreamConnected = true
			c.initiated = true
		case <-c.refreshTicker.C:
			if c.polling {
				if err := c.fetchFeatures(); err != nil {
					c.logger.Printf("Error refreshing features - %s", err.Error())
				}
			}
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
	go func() {
		if _, err := c.httpClient.Do(req); err != nil {
			c.logger.Printf("Error uploading event to analytics HTTP endpoint - %s", err.Error())
		}
	}()
	return nil
}

type eventOptions struct {
	FeatureID   string                 `json:"featureId"`
	UserID      string                 `json:"userId"`
	FeatureName string                 `json:"featureName"`
	Event       string                 `json:"event"`
	Tags        map[string]interface{} `json:"tags"`
	TestType    string                 `json:"testType"`
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
