package main

import (
	"fmt"
	"os"

	"github.com/molassessapp/molasses-go"
)

func main() {
	client, err := molasses.Init(molasses.ClientOptions{
		APIKey: os.Getenv("MOLASSES_API_KEY"),
	})

	if err != nil {
		fmt.Println(err.Error())
	}

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
