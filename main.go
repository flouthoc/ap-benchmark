package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/mattn/go-mastodon"
	"os"
	"strconv"
	"sync"
	"time"
)

func createToots(client *mastodon.Client, deleteToot bool) {
	start := time.Now()
	status, err := client.PostStatus(context.Background(), &mastodon.Toot{
		Status: "Setting sample status",
	})
	if err != nil {
		fmt.Printf("Error %w while setting the status/toot\n", err)
		panic(err)
	}
	t := time.Now()
	fmt.Printf("Duration: %+v to post a status\n", t.Sub(start))

	fmt.Println("Posted status id is: " + status.ID)

	if deleteToot {
		start = time.Now()
		// delete the newly created status
		client.DeleteStatus(context.Background(), status.ID)
		t = time.Now()
		fmt.Printf("Duration: %+v to delete a status \n", t.Sub(start))
	}
}

func main() {
	var wg sync.WaitGroup

	clientId := flag.String("client-id", "", "Activitypub client id")
	clientSecret := flag.String("client-secret", "", "Activitypub client secret")
	instance := flag.String("instance", "", "Activitypub instance")
	accessToken := flag.String("access-token", "", "Activitypub access token")
	load := flag.String("load", "", "Numer of requests to run")
	parallel := flag.String("parallel", "", "Parallel request or sequential")

	flag.Parse()

	if *clientId == "" || *clientSecret == "" || *instance == "" || *accessToken == "" {
		fmt.Println("All flags must be set, please --help for info")
		os.Exit(3)
	}

	client := mastodon.NewClient(&mastodon.Config{
		Server:       *instance,
		ClientID:     *clientId,
		ClientSecret: *clientSecret,
		AccessToken:  *accessToken,
	})

	timeline, err := client.GetTimelineHome(context.Background(), nil)
	if err != nil {
		fmt.Println("Error while fetching the timeline:\n")
		panic(err)
	}
	for i := len(timeline) - 1; i >= 0; i-- {
		fmt.Println(timeline[i])
	}

	// by default number of request should be 1.
	numberRequests := 1
	if *load != "" {
		parseLoad, err := strconv.Atoi(*load)
		if err != nil {
			panic(fmt.Sprintf("Unable to parse load requests %s: %w", load, err))
		}
		numberRequests = parseLoad
	}
	parallelRequests := false
	if *parallel != "" {
		parallelRequests = true
	}

	if parallelRequests {
		for j := 0; j < numberRequests; j++ {
			wg.Add(1)
			go func() {
				createToots(client, true)
				defer wg.Done()
			}()
		}
		wg.Wait()
	} else {
		for j := 0; j < numberRequests; j++ {
			createToots(client, true)
		}
	}
}
