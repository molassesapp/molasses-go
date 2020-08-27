package molasses_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/molassesapp/molasses-go"
	"github.com/stretchr/testify/assert"
)

func TestInitWithValidFeatureAndStop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/get-features", req.URL.String())

		rw.Write([]byte(`{"data":{"name":"Production","updatedAt":"2020-08-26T02:11:44Z","features":[{"id":"f603f621-83ba-46f0-adf5-70ed2d668646","key":"GOOGLE_SSO","description":"asdfasdf","active":true,"segments":[{"segmentType":"everyoneElse","userConstraints":[{"operator":"all","values":"","userParam":"","userParamType":""}],"percentage":100}]},{"id":"f3fae17d-a8d2-446f-8e85-bfa408562b73","key":"MOBILE_CHECKOUT","description":"asdfasd","active":false,"segments":[{"segmentType":"everyoneElse","userConstraints":[{"operator":"all","values":"","userParam":"","userParamType":""}],"percentage":100}]},{"id":"d1f276ce-80b7-45a7-a70d-dad190abcd6e","key":"NEW_CHECKOUT","description":"this is the new checkout screen","active":false,"segments":[{"segmentType":"everyoneElse","userConstraints":[{"operator":"all","values":"","userParam":"","userParamType":""}],"percentage":100}]}]}}`))
	}))

	client, err := molasses.Init(molasses.ClientOptions{
		HTTPClient: server.Client(),
		APIKey:     "API_KEY",
		URL:        server.URL,
	})
	assert.True(t, client.IsInitiated())
	if err != nil {
		t.Error(err)
	}
	assert.True(t, client.IsActive("GOOGLE_SSO"))
	assert.False(t, client.IsActive("MOBILE_CHECKOUT"))
	client.Stop()
	assert.False(t, client.IsInitiated())
}
