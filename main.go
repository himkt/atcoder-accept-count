package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

type Submission struct {
	Id          int    `json:"id"`
	ContestId   int    `json:"contest_id"`
	Result      string `json:"result"`
	Language    string `json:"language"`
	EpochSecond int64  `json:"epoch_second"`
}

type TwitterCredential struct {
	ConsumerKey    string
	ConsumerSecret string
	AccessToken    string
	AccessSecret   string
}

func tweetWrapper(client *twitter.Client, text string, useTwitterApi bool) {
	log.Println("USE API", useTwitterApi)

	if useTwitterApi {
		_, _, err := client.Statuses.Update(text, nil)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("[dry-run]")
		fmt.Println(text)
	}
}

func main() {
	jst, _ := time.LoadLocation("Asia/Tokyo")

	atcoderUserName := os.Getenv("ATCODER_USER")
	targetDate := os.Getenv("TARGET_DATE")
	credential := TwitterCredential{
		ConsumerKey:    os.Getenv("API_KEY"),
		ConsumerSecret: os.Getenv("API_KEY_SECRET"),
		AccessToken:    os.Getenv("ACCESS_TOKEN"),
		AccessSecret:   os.Getenv("ACCESS_TOKEN_SECRET"),
	}
	config := oauth1.NewConfig(credential.ConsumerKey, credential.ConsumerSecret)
	token := oauth1.NewToken(credential.AccessToken, credential.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)
	useTwitterApi := os.Getenv("USE_TWITTER_API") != ""

	currentTime := time.Now().In(jst)
	y, m, d := currentTime.Date()
	if targetDate != "" {
		parsedTargetDate, err := time.Parse("2006-01-02", targetDate)
		if err != nil {
			log.Fatal(err)
		}
		y, m, d = parsedTargetDate.In(jst).AddDate(0, 0, 1).Date()
	}
	todayBeginningOfDate := time.Date(y, m, d, 0, 0, 0, 0, jst)
	yesterdayBeginningOfDate := todayBeginningOfDate.AddDate(0, 0, -1)

	baseUrl := "https://kenkoooo.com/atcoder/atcoder-api/v3/user/submissions"
	apiUrl := fmt.Sprintf(
		"%s?user=%s&from_second=%d",
		baseUrl,
		atcoderUserName,
		yesterdayBeginningOfDate.Unix(),
	)
	log.Println("endpoint=", apiUrl)
	log.Printf("from=%v, to=%v", yesterdayBeginningOfDate, todayBeginningOfDate)

	response, err := http.Get(apiUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var submissions []Submission
	json.Unmarshal(body, &submissions)

	var acceptCounts int = 0
	for _, submission := range submissions {
		// NOTE(himkt)
		// Filter out submissions that are out of query date.
		// Needed since "/v3/user/submissions" does not support "to_second".
		if todayBeginningOfDate.Unix() <= submission.EpochSecond {
			continue
		}
		timestamp := time.Unix(submission.EpochSecond, 0).In(jst)
		log.Printf("unix=%d, timestamp=%v, result=%s", submission.EpochSecond, timestamp, submission.Result)
		if submission.Result == "AC" {
			acceptCounts++
		}
	}

	tweetText := fmt.Sprintf(`Date: %d/%d/%d
User: %s
AC count: %d

#atcoder_shojin`,
		y,
		m,
		d,
		atcoderUserName,
		acceptCounts,
	)
	tweetWrapper(client, tweetText, useTwitterApi)
}
