package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"os"
	"os/exec"
)

func LoadClientConfig() (*oauth1a.ClientConfig, error) {
	credentials, err := ioutil.ReadFile("CREDENTIALS")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(credentials), "\n")
	return &oauth1a.ClientConfig{
		ConsumerKey:    lines[0],
		ConsumerSecret: lines[1],
		CallbackURL:	"oob",
	}, nil
}

func GetAuthorizeURL(baseURL string, screenName string) string {
	screenName = url.QueryEscape(screenName)
	return baseURL + "&force_login=true" + "&screen_name=" + screenName
}

func CreateTwitterClient(id string) *twittergo.Client {
	clientConfig, err := LoadClientConfig()
	ErrorHandler(err)

	service := &oauth1a.Service{
		RequestURL:   "https://api.twitter.com/oauth/request_token",
		AuthorizeURL: "https://api.twitter.com/oauth/authenticate",
		AccessURL:    "https://api.twitter.com/oauth/access_token",
		ClientConfig: clientConfig,
		Signer: new(oauth1a.HmacSha1Signer),
	}

	httpClient := new(http.Client)
	userConfig := &oauth1a.UserConfig{}
	err = userConfig.GetRequestToken(service, httpClient)
	ErrorHandler(err)

	baseURL, err := userConfig.GetAuthorizeURL(service)
	ErrorHandler(err)

	authorizeURL := GetAuthorizeURL(baseURL, id)
	var (
		token string
		verifier string
	)
	fmt.Println("Open this URL and enter PIN.")
	fmt.Println(authorizeURL)

	browser := "xdg-open"
	args := []string{authorizeURL}

	browser, err = exec.LookPath(browser)
	if err == nil {
		cmd := exec.Command(browser, args...)
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		ErrorHandler(err)
	}

	fmt.Print("PIN: ")
	stdin := bufio.NewScanner(os.Stdin)
	if !stdin.Scan() {
		fmt.Errorf("canceled")
		os.Exit(1)
	}

	token = userConfig.RequestTokenKey
	verifier = stdin.Text()
	err = userConfig.GetAccessToken(token, verifier, service, httpClient)
	ErrorHandler(err)

	return twittergo.NewClient(clientConfig, userConfig)
}

func LoadCommand() string {
	fmt.Print("Type \"tweet\" or \"show\" or type \"exit\"\n")
	fmt.Print("Select> ")
	stdin := bufio.NewScanner(os.Stdin)
	if !stdin.Scan() {
		fmt.Errorf("canceled")
		os.Exit(1)
	}
	return stdin.Text()
}

func LoadStatus() string {
	fmt.Print("Tweet status> ")
	stdin := bufio.NewScanner(os.Stdin)
	if !stdin.Scan() {
		fmt.Errorf("canceled")
		os.Exit(1)
	}
	return stdin.Text()
}

func SendTweet(client *twittergo.Client) {
	data := url.Values{}
	data.Set("status", LoadStatus())
	body := strings.NewReader(data.Encode())
	req, err := http.NewRequest("POST", "/1.1/statuses/update.json", body)
	ErrorHandler(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.SendRequest(req)
	ErrorHandler(err)
}

func ShowTimeline(client *twittergo.Client) {
	req, err := http.NewRequest("GET", "/1.1/statuses/home_timeline.json", nil)
	ErrorHandler(err)
	resp, err := client.SendRequest(req)
	ErrorHandler(err)
	results := &twittergo.Timeline{}
	err = resp.Parse(results)
	ErrorHandler(err)
	for _, tweet := range *results {
		fmt.Printf("%v: %v\n", tweet.User().Name(), tweet.Text())
	}
}

func ErrorHandler(err error) {
	if err != nil {
		log.Printf("%v\n", err)
		os.Exit(1)
	}
}

func main() {
	var id string
	flag.StringVar(&id, "id", "", "ID")
	flag.Parse()

	client := CreateTwitterClient(id)

	for ;; {
		command := LoadCommand()
		if command == "tweet" {
			SendTweet(client)
		} else if command == "show" {
			ShowTimeline(client)
		} else if command == "exit" {
			os.Exit(0)
		}
	}
}
