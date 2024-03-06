package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/mattn/go-mastodon"
	"os"
	"time"
)

func main() {
	clientId := flag.String("client-id", "", "Activitypub client id")
	clientSecret := flag.String("client-secret", "", "Activitypub client secret")
	instance := flag.String("instance", "", "Activitypub instance")
	accessToken := flag.String("access-token", "", "Activitypub access token")

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

	start = time.Now()
	// delete the newly created status
	client.DeleteStatus(context.Background(), status.ID)
	t = time.Now()
	fmt.Printf("Duration: %+v to delete a status \n", t.Sub(start))
}
