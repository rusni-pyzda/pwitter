package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/rusni-pyzda/pwitter/tweetdiff"
	"log"
	"os"

	"github.com/Ukraine-DAO/twitter-threads/common"
	"github.com/Ukraine-DAO/twitter-threads/twitter"
	"github.com/rusni-pyzda/pwitter"
)

var (
	rcfg = common.RequestConfig{
		Expansions: []string{
			"author_id",
			"attachments.media_keys",
			"referenced_tweets.id",
			"referenced_tweets.id.author_id",
		},
		TweetFields: []string{
			"author_id",
			"conversation_id",
			"entities",
			"referenced_tweets",
			"text",
			"attachments",
			"created_at",
			"in_reply_to_user_id",
		},
		MediaFields: []string{
			"media_key",
			"type",
			"url",
			"preview_image_url",
			"variants",
			"alt_text",
		},
	}
)

func main() {
	flag.Parse()

	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <tweet_id>", os.Args[0])
	}

	id := os.Args[1]

	public, err := twitter.FetchTweet(id, rcfg)
	if err != nil {
		log.Fatalf("Failed to fetch the tweet using public API: %s", err)
	}

	auth, err := pwitter.AnonymousAuth(context.Background())
	if err != nil {
		log.Fatalf("constructing authorizer: %s", err)
	}
	client := &pwitter.Client{Authorizer: auth}

	private, err := client.TweetDetail(context.Background(), id)
	if err != nil {
		log.Fatalf("Failed to fetch the tweet using private API: %s", err)
	}

	diff := tweetdiff.Diff(&public, &private.Tweet)
	if diff != "" {
		fmt.Println(diff)
		os.Exit(1)
	}
}
