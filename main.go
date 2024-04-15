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

	clientIdSecond := flag.String("client-id-second", "", "Activitypub client id")
	clientSecretSecond := flag.String("client-secret-second", "", "Activitypub client secret")
	instanceSecond := flag.String("instance-second", "", "Activitypub instance")
	accessTokenSecond := flag.String("access-token-second", "", "Activitypub access token")

	userIdFirst := flag.String("userid-first", "", "First User Id")
	userIdSecond := flag.String("userid-second", "", "Second User Id")
	load := flag.String("load", "", "Numer of requests to run")
	parallel := flag.String("parallel", "", "Parallel request or sequential")

	flag.Parse()

	if *clientId == "" || *clientSecret == "" || *instance == "" || *accessToken == "" {
		fmt.Println("All flags must be set, please --help for info")
		os.Exit(3)
	}

	if *clientIdSecond != "" || *clientSecretSecond != "" || *instanceSecond != "" || *accessTokenSecond != "" {
		if *clientIdSecond == "" || *clientSecretSecond == "" || *instanceSecond == "" || *accessTokenSecond == "" {
			fmt.Println("All flags for second instance must be set, please --help for info")
			os.Exit(3)
		}
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

	if *userIdSecond != "" {
		clientSecond := mastodon.NewClient(&mastodon.Config{
			Server:       *instanceSecond,
			ClientID:     *clientIdSecond,
			ClientSecret: *clientSecretSecond,
			AccessToken:  *accessTokenSecond,
		})

		accounts, err := clientSecond.Search(context.Background(), *userIdFirst, true)
		if err != nil {
			fmt.Printf("unable to get the account %s:\n", *userIdFirst)
			panic(err)
		}

		if len(accounts.Accounts) > 0 {
			fmt.Printf("Successfully got account now tryiing to follow\n")
			_, err := clientSecond.AccountFollow(context.Background(), accounts.Accounts[0].ID)
			if err != nil {
				fmt.Printf("unable to follow the account %s\n", *userIdFirst)
				panic(err)
			}
		}

		accounts, err = client.Search(context.Background(), *userIdSecond, true)
		if err != nil {
			fmt.Printf("unable to get the account %s:\n", *userIdSecond)
			panic(err)
		}

		time.Sleep(1 * time.Second)

		err = client.FollowRequestAuthorize(context.Background(), accounts.Accounts[0].ID)
		if err != nil {
			fmt.Printf("unable accept follow request %s:\n", *userIdSecond)
			panic(err)
		}

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
				createToots(client, false)
				defer wg.Done()
			}()
		}
		wg.Wait()
	} else {
		for j := 0; j < numberRequests; j++ {
			createToots(client, false)
		}
	}

	timeline, err = client.GetTimelinePublic(context.Background(), false, nil)
	if err != nil {
		fmt.Println("Error while fetching the timeline:\n")
		panic(err)
	}
	fmt.Printf("Number of tweets in first user's public timeline %+v\n", len(timeline))
	timeline, err = client.GetTimelineHome(context.Background(), nil)
	if err != nil {
		fmt.Println("Error while fetching the timeline:\n")
		panic(err)
	}
	fmt.Printf("Number of tweets in first user's home timeline %+v\n", len(timeline))

}
