<p align="center">
<img src="https://raw.githubusercontent.com/molassesapp/molasses-go/main/logo.png" style="margin: 0px auto;" width="200"/></p>

<h1 align="center">Molasses-Go</h1>

[![codecov](https://codecov.io/gh/molassesapp/molasses-go/branch/main/graph/badge.svg)](https://codecov.io/gh/molassesapp/molasses-go)
![Build status](https://github.com/molassesapp/molasses-go/workflows/Go/badge.svg)

A Go SDK for Molasses. It allows you to evaluate a user's status for a feature. It also helps simplify logging events for A/B testing.

Molasses uses Server Sent Events for instant updates to feature flags. Once initialized, it takes microseconds to evaluate if a user is active. When you update a feature flag on Molasses all of your clients are instantly updated

## Install

```
go get github.com/molassesapp/molasses-go
```

## Usage

### Initialization

Start by initializing the client with an `APIKey`. This begins the polling for any feature updates. The updates happen every 15 seconds.

```go
	client, err := molasses.Init(molasses.ClientOptions{
		APIKey: os.Getenv("MOLASSES_API_KEY"),
	})
```

If you decide not to track analytics events (experiment started, experiment success) you can turn them off by setting the `SendEvents` field to `molasses.bool(false)`

```go
	client, err := molasses.Init(molasses.ClientOptions{
		SendEvents: molasses.Bool(false),
		APIKey:     os.Getenv("MOLASSES_API_KEY"),
	})
```

### Check if feature is active

You can call `isActive` with the key name and optionally a user's information. The ID field is used to determine whether a user is part of a percentage of users. If you have other constraints based on user params you can pass those in the `Params` field.

```go
client.IsActive("TEST_FEATURE_FOR_USER", molasses.User{
		ID: "baz",
		Params: map[string]string{
			"teamId": "12356",
		},
	})
```

You can check if a feature is active for a user who is anonymous by just calling `isActive` with the key. You won't be able to do percentage roll outs or track that user's behavior.

```go
client.IsActive("TEST_FEATURE_FOR_USER")
```

### Experiments

To track whether an experiment was successful you can call `ExperimentSuccess`. ExperimentSuccess takes the feature's name, the molasses User and any additional parameters for the event.

```go
client.ExperimentSuccess("GOOGLE_SSO", molasses.User{
		ID: "baz",
		Params: map[string]string{
			"teamId": "12356",
		},
	}, map[string]string{
		"version": "v2.3.0"
	})
```

## Example

```go

import (
	"fmt"
	"os"

	"github.com/molassessapp/molasses-go"
)

func main() {
	client, err := molasses.Init(molasses.ClientOptions{
		APIKey: os.Getenv("MOLASSES_API_KEY"),
	})

	if client.IsActive("New Checkout") {
		fmt.Println("we are a go")
	} else {
		fmt.Println("we are a no go")
	}

	if client.IsActive("Another feature", molasses.User{
		ID: "baz",
		Params: map[string]string{
			"teamId": "12356",
		},
	}) {
		fmt.Println("we are a go")
	} else {
		fmt.Println("we are a no go")
	}
}
```
