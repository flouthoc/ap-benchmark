package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/mattn/go-mastodon"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

const METRICS_PATH = "/tmp/tweet_fan_out_metrics"

func init() {
	rand.Seed(time.Now().UnixNano())
}

type AccountCreateRequest struct {
	// Text that will be reviewed by moderators if registrations require manual approval.
	Reason string `form:"reason" json:"reason" xml:"reason"`
	// The desired username for the account.
	// swagger:parameters
	// pattern: [a-z0-9_]{2,64}
	// example: a_valid_username
	// required: true
	Username string `form:"username" json:"username" xml:"username" binding:"required"`
	// The email address to be used for login.
	// swagger:parameters
	// example: someone@wherever.com
	// required: true
	Email string `form:"email" json:"email" xml:"email" binding:"required"`
	// The password to be used for login. This will be hashed before storage.
	// swagger:parameters
	// example: some_really_really_really_strong_password
	// required: true
	Password string `form:"password" json:"password" xml:"password" binding:"required"`
	// The user agrees to the terms, conditions, and policies of the instance.
	// swagger:parameters
	// required: true
	Agreement bool `form:"agreement"  json:"agreement" xml:"agreement" binding:"required"`
	// The language of the confirmation email that will be sent.
	// swagger:parameters
	// example: en
	// Required: true
	Locale string `form:"locale" json:"locale" xml:"locale" binding:"required"`
}

type Follower struct {
	Instance     string
	AccessToken  string
	UserId       string
	ClientId     string
	ClientSecret string
}

func main() {
	var wg sync.WaitGroup

	instance := flag.String("instance", getEnv("SERVER_URL", ""), "Activitypub first instance")
	instanceSecond := flag.String("instance-second", getEnv("SERVER_URL_SECOND", ""), "Activitypub second instance")
	followersLocal := flag.Int("followers-local", 0, "No of local followers ( followers on same instance )")
	followersFederated := flag.Int("followers-fed", 1, "No of federated followers ( followers on second instance )")
	load := flag.Int("load", 1, "Numer of requests to run")
	parallel := flag.Bool("parallel", false, "Parallel request or sequential")
	deleteToots := flag.Bool("delete-toots", false, "Delete toots when test ends")
	showOutputGraph := flag.Bool("show-graph", false, "Show output graph")
	remoteMetricsPath := flag.String("remote-metrics", "", "Remote metrics URL")
	outputFileName := ""

	flag.Parse()

	if *instance == "" {
		fmt.Println("All flags must be set, please --help for info")
		os.Exit(3)
	}

	username := generateRandomString(10)
	accessToken, userIdFirst, clientId, clientSecret, err := createUser(username, *instance)
	if err != nil {
		panic(fmt.Sprintf("Failed creating user: %+v\n", err))
	}

	client := mastodon.NewClient(&mastodon.Config{
		Server:       *instance,
		ClientID:     clientId,
		ClientSecret: clientSecret,
		AccessToken:  accessToken,
	})

	timeline, err := client.GetTimelineHome(context.Background(), nil)
	if err != nil {
		fmt.Println("Error while fetching the timeline:\n")
		panic(err)
	}
	for i := len(timeline) - 1; i >= 0; i-- {
		fmt.Println(timeline[i])
	}

	if *instanceSecond != "" {
		followers := []Follower{}
		for i := 0; i < *followersFederated; i++ {
			followers = append(followers,
				Follower{Instance: *instanceSecond})
		}
		createAndAcceptFollowers(client, userIdFirst, followers)
	}

	if *followersLocal > 0 {
		followers := []Follower{}
		for i := 0; i < *followersLocal; i++ {
			followers = append(followers,
				Follower{Instance: *instance})
		}
		createAndAcceptFollowers(client, userIdFirst, followers)
	}

	numberRequests := *load
	parallelRequests := *parallel
	postResults := []time.Duration{}
	deleteResults := []time.Duration{}

	if parallelRequests {
		for j := 0; j < numberRequests; j++ {
			wg.Add(1)
			go func() {
				postDuration, deleteDuration := createToots(client, *deleteToots)
				postResults = append(postResults, postDuration)
				deleteResults = append(deleteResults, deleteDuration)
				defer wg.Done()
			}()
		}
		wg.Wait()
		if *instanceSecond == "" {
			points := durationToPlotters(postResults)
			plotGraph("parallel", points)
			outputFileName = "parallel.png"
		} else {
			if *remoteMetricsPath != "" {
				err := DownloadFile(METRICS_PATH, *remoteMetricsPath)
				if err != nil {
					fmt.Println("Error while downloading raw metrics from remote server")
					panic(err)
				}
			}
			points := durationToPlotters(readMetrics(METRICS_PATH))
			plotGraph("parallel-federated", points)
			outputFileName = "parallel-federated.png"

		}
	} else {
		for j := 0; j < numberRequests; j++ {
			postDuration, deleteDuration := createToots(client, *deleteToots)
			postResults = append(postResults, postDuration)
			deleteResults = append(deleteResults, deleteDuration)

		}
		if *instanceSecond == "" {
			points := durationToPlotters(postResults)
			plotGraph("sequential", points)
			outputFileName = "sequential.png"
		} else {
			err := DownloadFile(METRICS_PATH, *remoteMetricsPath)
			if err != nil {
				fmt.Println("Error while downloading raw metrics from remote server")
				panic(err)
			}
			points := durationToPlotters(readMetrics(METRICS_PATH))
			plotGraph("sequential-federated", points)
			outputFileName = "sequential-federated.png"
		}
	}

	timeline, err = client.GetTimelinePublic(context.Background(), false, nil)
	if err != nil {
		fmt.Println("Error while fetching the timeline:\n")
		panic(err)
	}
	fmt.Printf("Number of tweets in first user's public timeline %+v\n\n", len(timeline))
	timeline, err = client.GetTimelineHome(context.Background(), nil)
	if err != nil {
		fmt.Println("Error while fetching the timeline:\n")
		panic(err)
	}
	fmt.Printf("Number of tweets in first user's home timeline %+v\n\n", len(timeline))
	fmt.Printf("Post durations %+v\n", postResults)
	fmt.Printf("Metrics graph generated at %s\n", outputFileName)

	if *showOutputGraph {
		//runCommand("xdg-open", outputFileName)
	}
}

func DownloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func readMetrics(path string) []time.Duration {
	result := []time.Duration{}
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		fmt.Println(scanner.Text())
		line := scanner.Text()
		split := strings.Split(line, " ")
		durationStr := split[len(split)-1]
		parsedDuration, err := time.ParseDuration(durationStr)
		if err != nil {
			panic(fmt.Sprintf("Unable to parse duration from metrics %+v\n", err))
		}
		result = append(result, parsedDuration)
	}

	if err := scanner.Err(); err != nil {
		panic(fmt.Sprintf("Unable to parse metrics %+v\n", err))
	}
	return result
}

func createAndAcceptFollowers(parentUserClient *mastodon.Client, parentUserId string, followers []Follower) {
	for _, follower := range followers {

		username := generateRandomString(10)
		accessTokenSecond, userIdSecond, clientIdSecond, clientSecretSecond, err := createUser(username, follower.Instance)
		if err != nil {
			panic(fmt.Sprintf("Failed creating user: %+v\n", err))
		}
		follower.UserId = userIdSecond
		follower.AccessToken = accessTokenSecond
		follower.ClientId = clientIdSecond
		follower.ClientSecret = clientSecretSecond

		clientSecond := mastodon.NewClient(&mastodon.Config{
			Server:       follower.Instance,
			ClientID:     follower.ClientId,
			ClientSecret: follower.ClientSecret,
			AccessToken:  follower.AccessToken,
		})

		accounts, err := clientSecond.Search(context.Background(), parentUserId, true)
		if err != nil {
			fmt.Printf("unable to get the account %s:\n", parentUserId)
			panic(err)
		}

		if len(accounts.Accounts) > 0 {
			fmt.Printf("Successfully got account now tryiing to follow\n")
			_, err := clientSecond.AccountFollow(context.Background(), accounts.Accounts[0].ID)
			if err != nil {
				fmt.Printf("unable to follow the account %s\n", parentUserId)
				panic(err)
			}
		}
		// If test fails with 404 while following a user, increase this timeout
		time.Sleep(50 * time.Millisecond)

		accounts, err = parentUserClient.Search(context.Background(), follower.UserId, true)
		if err != nil {
			fmt.Printf("unable to get the account %s:\n", follower.UserId)
			panic(err)
		}

		err = parentUserClient.FollowRequestAuthorize(context.Background(), accounts.Accounts[0].ID)
		if err != nil {
			fmt.Printf("unable accept follow request %s and total accounts %+v:\n", follower.UserId, accounts.Accounts[0])
			panic(err)
		}
	}
}

func createToots(client *mastodon.Client, deleteToot bool) (time.Duration, time.Duration) {
	start := time.Now()
	status, err := client.PostStatus(context.Background(), &mastodon.Toot{
		Status: "Setting sample status",
	})
	if err != nil {
		fmt.Printf("Error %w while setting the status/toot\n", err)
		panic(err)
	}
	t := time.Now()
	postDuration := t.Sub(start)
	fmt.Printf("Duration: %+v to post a status\n", postDuration)

	fmt.Println("Posted status id is: " + status.ID)

	var deleteDuration time.Duration
	if deleteToot {
		start = time.Now()
		// delete the newly created status
		client.DeleteStatus(context.Background(), status.ID)
		t = time.Now()
		deleteDuration = t.Sub(start)
		fmt.Printf("Duration: %+v to delete a status \n", postDuration, deleteDuration)
	}
	return postDuration, deleteDuration
}

func createUser(username, serverURL string) (string, string, string, string, error) {
	domain := strings.Split(strings.Split(serverURL, "://")[1], "/")[0]
	redirectURI := serverURL
	clientName := "Test Application Name"
	registrationReason := "Testing whether or not this dang diggity thing works!"
	registrationUsername := username
	registrationEmail := fmt.Sprintf("%s@%s", username, domain)
	registrationPassword := "very good password 123"
	registrationAgreement := true
	registrationLocale := "en"

	// Step 1: create the app to register the new account
	createAppURL := fmt.Sprintf("%s/api/v1/apps", serverURL)
	createAppData := map[string]string{
		"client_name":   clientName,
		"redirect_uris": redirectURI,
	}
	createAppResponse := makePostRequest(createAppURL, createAppData)
	clientID := createAppResponse["client_id"].(string)
	clientSecret := createAppResponse["client_secret"].(string)
	fmt.Printf("Obtained client_id: %s and client_secret: %s\n", clientID, clientSecret)

	// Step 2: obtain a code for that app
	appCodeURL := fmt.Sprintf("%s/oauth/token", serverURL)
	appCodeData := map[string]string{
		"scope":         "read",
		"grant_type":    "client_credentials",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"redirect_uri":  redirectURI,
	}
	appCodeResponse := makePostRequest(appCodeURL, appCodeData)
	appAccessToken := appCodeResponse["access_token"].(string)
	fmt.Printf("Obtained app access token: %s\n", appAccessToken)
	accountRegisterData := AccountCreateRequest{
		Reason:    registrationReason,
		Email:     registrationEmail,
		Username:  registrationUsername,
		Password:  registrationPassword,
		Agreement: registrationAgreement,
		Locale:    registrationLocale,
	}
	accountRegisterHeaders := map[string]string{
		"Content-Type":  "application/json; charset=UTF-8",
		"Authorization": fmt.Sprintf("Bearer %s", appAccessToken),
	}
	// Step 3: use the code to register a new account
	accountRegisterURL := fmt.Sprintf("%s/api/v1/accounts", serverURL)
	accountRegisterDataJson, err := json.Marshal(accountRegisterData)
	if err != nil {
		return "", "", "", "", err
	}
	accountRegisterResponse := makePostRequestWithHeaders(accountRegisterURL, accountRegisterDataJson, accountRegisterHeaders)
	fmt.Printf("Obtained user access token: %+v\n", accountRegisterResponse)
	userAccessToken := accountRegisterResponse["access_token"].(string)
	fmt.Printf("Obtained user access token: %s\n", userAccessToken)

	// Step 4: verify the returned access token
	verifyCredentialsURL := fmt.Sprintf("%s/api/v1/accounts/verify_credentials", serverURL)
	verifyResponse := makeGetRequestWithHeaders(verifyCredentialsURL, map[string]string{"Authorization": fmt.Sprintf("Bearer %s", userAccessToken)})
	fmt.Println(string(verifyResponse))

	return userAccessToken, registrationEmail, clientID, clientSecret, nil
}

func runCommand(command string, args ...string) {
	cmd := exec.Command(command, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Error running command: %s", err)
	}
	fmt.Println(out.String())
}

func getEnv(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		value = defaultValue
	}
	return value
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func generateRandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func makePostRequest(url string, data map[string]string) map[string]interface{} {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Error encoding JSON: %s", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error making POST request: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %s", err)
	}
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatalf("Error decoding JSON response: %s", err)
	}
	return result
}

func makePostRequestWithHeaders(url string, data []byte, headers map[string]string) map[string]interface{} {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Fatalf("Error creating POST request: %s", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error making POST request with headers: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %s", err)
	}
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatalf("Error decoding JSON response: %s", err)
	}
	return result
}

func makeGetRequestWithHeaders(url string, headers map[string]string) []byte {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Error creating GET request: %s", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error making GET request with headers: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %s", err)
	}
	return body
}

func plotGraph(name string, durations plotter.XYs) {
	p := plot.New()

	p.Title.Text = name
	p.X.Label.Text = "Requests"
	p.Y.Label.Text = "Duration in ms"

	err := plotutil.AddLinePoints(p,
		"Requests", durations)
	if err != nil {
		panic(err)
	}

	// Save the plot to a PNG file.
	if err := p.Save(5*vg.Inch, 5*vg.Inch, name+".png"); err != nil {
		panic(err)
	}
}

func durationToPlotters(n []time.Duration) plotter.XYs {
	pts := make(plotter.XYs, len(n))
	for i, v := range n {
		pts[i].X = float64(i)
		pts[i].Y = float64(v) / float64(time.Millisecond)
	}
	return pts
}
