<p align="center">
<img src="https://raw.githubusercontent.com/molassesapp/molasses-go/main/logo.png" style="margin: 0px auto;" width="200"/></p>

<h1 align="center">Molasses-Go</h1>

[![codecov](https://codecov.io/gh/molassesapp/molasses-go/branch/main/graph/badge.svg)](https://codecov.io/gh/molassesapp/molasses-go)
![Build status](https://github.com/molassesapp/molasses-go/workflows/Go/badge.svg)

A Go SDK for Molasses. It allows you to evaluate user's status for a feature. It also helps simplify logging events for A/B testing.

Molasses uses polling to check if you have updated features. Once initialized, it takes microseconds to evaluate if a user is active.

## Install

```
go get github.com/molassesapp/molasses-go
```

## Usage

Start by initializing the client with an `APIKey`

```go
	client, err := molasses.Init(molasses.ClientOptions{
		APIKey: os.Getenv("MOLASSES_API_KEY"),
  })
```

Then you can call `isActive` with the key name and optionally a user's information

```go
client.IsActive("TEST_FEATURE_FOR_USER")

client.IsActive("TEST_FEATURE_FOR_USER", molasses.User{
		ID: "baz",
		Params: map[string]string{
			"teamId": "12356",
		},
	})
```

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

	if client.IsActive("TEST_FEATURE_FOR_USER") {
		fmt.Println("we are a go")
	} else {
		fmt.Println("we are a no go")
	}

	if client.IsActive("TEST_FEATURE_FOR_USER", molasses.User{
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
