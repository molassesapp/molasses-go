package molasses_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/molassesapp/molasses-go"
	"github.com/stretchr/testify/assert"
)

type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, errors.New("error http client mock")
}

func TestInitWithValidFeatureAndStop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/features", req.URL.String())

		if _, err := rw.Write([]byte(`{"data":{"name":"Production","updatedAt":"2020-08-26T02:11:44Z","features":[{"id":"f603f621-83ba-46f0-adf5-70ed2d668646","key":"GOOGLE_SSO","description":"asdfasdf","active":true,"segments":[{"segmentType":"everyoneElse","userConstraints":[{"operator":"all","values":"","userParam":"","userParamType":""}],"percentage":100}]},{"id":"f3fae17d-a8d2-446f-8e85-bfa408562b73","key":"MOBILE_CHECKOUT","description":"asdfasd","active":false,"segments":[{"segmentType":"everyoneElse","userConstraints":[{"operator":"all","values":"","userParam":"","userParamType":""}],"percentage":100}]},{"id":"d1f276ce-80b7-45a7-a70d-dad190abcd6e","key":"NEW_CHECKOUT","description":"this is the new checkout screen","active":false,"segments":[{"segmentType":"everyoneElse","userConstraints":[{"operator":"all","values":"","userParam":"","userParamType":""}],"percentage":100}]}]}}`)); err != nil {
			t.Error(err)
		}
	}))

	client, err := molasses.Init(molasses.ClientOptions{
		HTTPClient:     server.Client(),
		Polling:        true,
		APIKey:         "API_KEY",
		URL:            server.URL,
		AutoSendEvents: false,
	})
	assert.True(t, client.IsInitiated())
	if err != nil {
		t.Error(err)
	}
	assert.True(t, client.IsActive("GOOGLE_SSO"))
	assert.False(t, client.IsActive("MOBILE_CHECKOUT", molasses.User{ID: "USERID1"}))
	client.Stop()
	assert.False(t, client.IsInitiated())
}

func TestInitWithInvalidClientAndStop(t *testing.T) {
	server := httptest.NewServer(&http.ServeMux{})

	client, err := molasses.Init(molasses.ClientOptions{
		HTTPClient:     &MockClient{},
		Polling:        true,
		APIKey:         "API_KEY",
		URL:            server.URL,
		AutoSendEvents: false,
	})
	assert.False(t, client.IsInitiated())
	if err != nil {
		t.Error(err)
	}
	assert.False(t, client.IsActive("GOOGLE_SSO"))
	assert.False(t, client.IsActive("MOBILE_CHECKOUT", molasses.User{ID: "USERID1"}))
}

func TestDefaultsAreSet(t *testing.T) {
	client, err := molasses.Init(molasses.ClientOptions{
		APIKey:         "API_KEY",
		Polling:        true,
		AutoSendEvents: false,
	})
	assert.True(t, client.IsInitiated())
	if err != nil {
		t.Error(err)
	}
}

func TestErrorsWhenAPIKeyIsNotSet(t *testing.T) {
	_, err := molasses.Init(molasses.ClientOptions{
		APIKey:         "",
		Polling:        true,
		AutoSendEvents: false,
	})
	if err != nil {
		assert.Error(t, err)
	}
}

func TestInitWithValidFeatureWithUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/features" {

			assert.Equal(t, "/features", req.URL.String())
			if _, err := rw.Write([]byte(`{"data":{"name":"Production","updatedAt":"2020-08-26T02:11:44Z","features":[{"id":"f603f621-83ba-46f0-adf5-70ed2d668646","key":"GOOGLE_SSO","description":"asdfasdf","active":true,"segments":[{"segmentType":"everyoneElse","userConstraints":[{"operator":"all","values":"","userParam":"","userParamType":""}],"percentage":100}]},{"id":"f3fae17d-a8d2-446f-8e85-bfa408562b73","key":"MOBILE_CHECKOUT","description":"asdfasd","active":false,"segments":[{"segmentType":"everyoneElse","userConstraints":[{"operator":"all","values":"","userParam":"","userParamType":""}],"percentage":100}]},{"id":"d1f276ce-80b7-45a7-a70d-dad190abcd6e","key":"NEW_CHECKOUT","description":"this is the new checkout screen","active":false,"segments":[{"segmentType":"everyoneElse","userConstraints":[{"operator":"all","values":"","userParam":"","userParamType":""}],"percentage":100}]}]}}`)); err != nil {
				t.Error(err)
			}
			return
		}
		assert.Equal(t, "/analytics", req.URL.String())
		if _, err := rw.Write([]byte(`{}`)); err != nil {
			t.Error(err)
		}
	}))

	client, err := molasses.Init(molasses.ClientOptions{
		HTTPClient: server.Client(),
		Polling:    true,
		APIKey:     "API_KEY",
		URL:        server.URL,
	})
	assert.True(t, client.IsInitiated())
	if err != nil {
		t.Error(err)
	}
	assert.True(t, client.IsActive("GOOGLE_SSO", molasses.User{
		ID: "1234",
		Params: map[string]interface{}{
			"foo": "bar",
		},
	}))
	assert.False(t, client.IsActive("MOBILE_CHECKOUT", molasses.User{
		ID: "1234",
		Params: map[string]interface{}{
			"foo": "bar",
		},
	}))
	client.Stop()
	time.Sleep(1 * time.Second)
	assert.False(t, client.IsInitiated())
}

func TestOtherSegments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/features" {

			assert.Equal(t, "/features", req.URL.String())
			if _, err := rw.Write([]byte(`{
				"data": {
					"name": "Production",
					"updatedAt": "2020-08-26T02:11:44Z",
					"features": [
						{
							"id": "f603f621-83ba-46f0-adf5-70ed2d668646",
							"key": "GOOGLE_SSO",
							"description": "asdfasdf",
							"active": true,
							"segments": [
								{
									"segmentType": "alwaysControl",
									"userConstraints": [
										{
											"operator": "equals",
											"values": "true",
											"userParam": "controlUser",
											"userParamType": ""
										},
										{
											"operator": "nin",
											"values": "yes,maybe,definitely",
											"userParam": "experimentUser",
											"userParamType": ""
										}
									],
									"percentage": 100
								},
								{
									"segmentType": "alwaysExperiment",
									"userConstraints": [
										{
											"operator": "contains",
											"values": "fals",
											"userParam": "controlUser",
											"userParamType": ""
										},
										{
											"operator": "in",
											"values": "yes,maybe,definitely",
											"userParam": "experimentUser",
											"userParamType": ""
										}
									],
									"percentage": 100
								},
								{
									"segmentType": "everyoneElse",
									"userConstraints": [
										{
											"operator": "all",
											"values": "",
											"userParam": "",
											"userParamType": ""
										}
									],
									"percentage": 50
								}
							]
						}
					]
				}
			}`)); err != nil {
				t.Error(err)
			}
			return
		}
		assert.Equal(t, "/analytics", req.URL.String())
		if _, err := rw.Write([]byte(`{}`)); err != nil {
			t.Error(err)
		}
	}))

	client, err := molasses.Init(molasses.ClientOptions{
		HTTPClient: server.Client(),
		Polling:    true,
		APIKey:     "API_KEY",
		URL:        server.URL,
	})
	assert.True(t, client.IsInitiated())
	if err != nil {
		t.Error(err)
	}
	experimentUser := molasses.User{
		ID: "1235",
		Params: map[string]interface{}{
			"controlUser":    "false",
			"experimentUser": "yes",
		},
	}
	controlUser := molasses.User{
		ID: "1234",
		Params: map[string]interface{}{
			"controlUser":    "true",
			"experimentUser": "nope",
		},
	}
	assert.False(t, client.IsActive("GOOGLE_SSO", controlUser))
	client.ExperimentSuccess("GOOGLE_SSO", controlUser, map[string]string{})
	assert.True(t, client.IsActive("GOOGLE_SSO", experimentUser))
	client.ExperimentSuccess("GOOGLE_SSO", experimentUser, map[string]string{})
	assert.False(t, client.IsActive("GOOGLE_SSO", molasses.User{
		ID: "1",
		Params: map[string]interface{}{
			"controlUser": "bar",
		},
	}))

	assert.True(t, client.IsActive("GOOGLE_SSO", molasses.User{
		ID: "2",
		Params: map[string]interface{}{
			"controlUser": "bar",
		},
	}))
	client.Stop()
	time.Sleep(1 * time.Second)
	assert.False(t, client.IsInitiated())
}

func TestMoreSegments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/features" {

			assert.Equal(t, "/features", req.URL.String())
			if _, err := rw.Write([]byte(`{
				"data": {
					"name": "Production",
					"updatedAt": "2020-08-26T02:11:44Z",
					"features": [
						{
							"id": "f603f621-83ba-46f0-adf5-70ed2d668646",
							"key": "GOOGLE_SSO",
							"description": "asdfasdf",
							"active": true,
							"segments": [
								{
									"segmentType": "alwaysControl",
									"userConstraints": [
										{
											"operator": "doesNotEqual",
											"values": "false",
											"userParam": "controlUser",
											"userParamType": ""
										},
										{
											"operator": "doesNotContain",
											"values": "yes",
											"userParam": "experimentUser",
											"userParamType": ""
										}
									],
									"percentage": 100
								},
								{
									"segmentType": "alwaysExperiment",
									"constraint":  "any",
									"userConstraints": [
										{
											"operator": "contains",
											"values": "fals",
											"userParam": "controlUser",
											"userParamType": ""
										},
										{
											"operator": "in",
											"values": "yes,maybe,definitely",
											"userParam": "experimentUser",
											"userParamType": ""
										},
										{
											"operator": "in",
											"values": "1235,123,1",
											"userParam": "id",
											"userParamType": ""
										}
									],
									"percentage": 100
								},
								{
									"segmentType": "everyoneElse",
									"userConstraints": [
										{
											"operator": "all",
											"values": "",
											"userParam": "",
											"userParamType": ""
										}
									],
									"percentage": 50
								}
							]
						},
						{
							"id": "f603f621-83ba-46f0-adf5-70ed2d668646",
							"key": "other_types",
							"description": "This tests a headline",
							"active": true,
							"segments": [
								{
									"segmentType": "alwaysControl",
									"userConstraints": [
										{
											"operator": "equals",
											"values": "true",
											"userParam": "controlUser",
											"userParamType": "bool"
										},
										{
											"operator": "doesNotEqual",
											"values": "true",
											"userParam": "experimentUser",
											"userParamType": "bool"
										}
									],
									"percentage": 100
								},
								{
									"segmentType": "alwaysExperiment",
									"constraint":  "any",
									"userConstraints": [
										{
											"operator": "equals",
											"values": 1235,
											"userParam": "userId",
											"userParamType": "number"
										},

										{
											"operator": "gte",
											"values": "14588.007",
											"userParam": "id",
											"userParamType": "number"
										}
									],
									"percentage": 100
								},
								{
									"segmentType": "everyoneElse",
									"userConstraints": [
										{
											"operator": "all",
											"values": "",
											"userParam": "",
											"userParamType": ""
										}
									],
									"percentage": 100
								}
								]
						},
						{
							"id": "f603f621-83ba-46f0-adf5-70ed2d668646",
							"key": "numbers",
							"description": "This tests a headline",
							"active": true,
							"segments": [
								{
									"segmentType": "alwaysControl",
									"userConstraints": [
										{
											"operator": "equals",
											"values": "500",
											"userParam": "controlUser",
											"userParamType": "number"
										},
										{
											"operator": "doesNotEqual",
											"values": "42",
											"userParam": "experimentUser",
											"userParamType": "number"
										},

										{
											"operator": "lt",
											"values": "14588.007",
											"userParam": "lt",
											"userParamType": "number"
										},

										{
											"operator": "lte",
											"values": "14588.007",
											"userParam": "lte",
											"userParamType": "number"
										}

									],
									"percentage": 100
								},
								{
									"segmentType": "alwaysExperiment",
									"constraint":  "any",
									"userConstraints": [
										{
											"operator": "gt",
											"values": "1235",
											"userParam": "userId",
											"userParamType": "number"
										}

									],
									"percentage": 100
								},
								{
									"segmentType": "everyoneElse",
									"userConstraints": [
										{
											"operator": "all",
											"values": "",
											"userParam": "",
											"userParamType": ""
										}
									],
									"percentage": 0
								}
							]
						}
					]
				}
			}`)); err != nil {
				t.Error(err)
			}
			return
		}
		assert.Equal(t, "/analytics", req.URL.String())
		if _, err := rw.Write([]byte(`{}`)); err != nil {
			t.Error(err)
		}
	}))

	client, err := molasses.Init(molasses.ClientOptions{
		HTTPClient: server.Client(),
		Polling:    true,
		APIKey:     "API_KEY",
		URL:        server.URL,
	})
	assert.True(t, client.IsInitiated())
	if err != nil {
		t.Error(err)
	}
	experimentUser := molasses.User{
		ID: "1235",
		Params: map[string]interface{}{
			"experimentUser": "yes",
		},
	}
	controlUser := molasses.User{
		ID: "1234",
		Params: map[string]interface{}{
			"controlUser":    "true",
			"experimentUser": "nope",
		},
	}
	assert.False(t, client.IsActive("GOOGLE_SSO", controlUser))
	client.ExperimentStarted("GOOGLE_SSO", controlUser, map[string]string{})
	client.Track("Checkout Started", controlUser, map[string]string{})
	client.ExperimentSuccess("GOOGLE_SSO", controlUser, map[string]string{})
	assert.True(t, client.IsActive("GOOGLE_SSO", experimentUser))
	client.ExperimentSuccess("GOOGLE_SSO", experimentUser, map[string]string{
		"experiment_id": "hello",
		"button_color":  "green",
	})
	assert.False(t, client.IsActive("GOOGLE_SSO", molasses.User{
		ID: "5",
		Params: map[string]interface{}{
			"controlUser": "bar",
		},
	}))

	assert.True(t, client.IsActive("GOOGLE_SSO", molasses.User{
		ID: "2",
		Params: map[string]interface{}{
			"controlUser": "bar",
		},
	}))

	experimentUser = molasses.User{
		ID: "1235",
		Params: map[string]interface{}{
			"userId": 1235,
		},
	}
	controlUser = molasses.User{
		ID: "1234",
		Params: map[string]interface{}{
			"controlUser":    true,
			"experimentUser": false,
		},
	}
	user1 := molasses.User{
		ID: "1236",
		Params: map[string]interface{}{
			"controlUser": false,
		},
	}
	assert.False(t, client.IsActive("other_types", controlUser))
	assert.True(t, client.IsActive("other_types", experimentUser))
	assert.True(t, client.IsActive("other_types", user1))
	assert.True(t, client.IsActive("other_types", molasses.User{
		ID: "500000",
	}))

	assert.True(t, client.IsActive("other_types", molasses.User{
		ID: "2",
		Params: map[string]interface{}{
			"controlUser": "bar",
		},
	}))
	assert.False(t, client.IsActive("numbers", molasses.User{
		ID: "2",
		Params: map[string]interface{}{
			"controlUser":    500,
			"experimentUser": 43,
			"lt":             -14580,
			"lte":            "14588.007",
		},
	}))
	assert.True(t, client.IsActive("numbers", molasses.User{
		ID: "2",
		Params: map[string]interface{}{
			"userId": 5235,
		},
	}))

	client.Stop()
	time.Sleep(1 * time.Second)
	assert.False(t, client.IsInitiated())
}
