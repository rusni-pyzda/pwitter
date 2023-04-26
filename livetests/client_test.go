package pwitter

import (
	"context"
	"net/http"
	"testing"

	"github.com/rs/zerolog"

	. "github.com/rusni-pyzda/pwitter"
)

const testAccountID = "783214" // https://twitter.com/Twitter

func createClient(ctx context.Context, t *testing.T) *Client {
	auth, err := AnonymousAuth(ctx)
	if err != nil {
		t.Fatalf("Failed to construct authorizer: %s", err)
	}

	return &Client{
		Authorizer: auth,
		Client:     http.DefaultClient,
	}
}

func TestUserTweets(t *testing.T) {
	out := zerolog.NewConsoleWriter(zerolog.ConsoleTestWriter(t))
	log := zerolog.New(out).Level(zerolog.DebugLevel)
	ctx := log.WithContext(context.Background())
	client := createClient(ctx, t)
	r1, err := client.UserTweets(ctx, testAccountID, "")
	if err != nil {
		t.Fatalf("UserTweets returned error: %s", err)
	}
	for _, tw := range r1.Tweets {
		t.Logf("%s", tw.Text)
	}
	r2, err := client.UserTweets(ctx, testAccountID, r1.CursorNext)
	if err != nil {
		t.Fatalf("UserTweets returned error: %s", err)
	}
	for _, tw := range r2.Tweets {
		t.Logf("%s", tw.Text)
	}
}

func TestTweetDetail(t *testing.T) {
	out := zerolog.NewConsoleWriter(zerolog.ConsoleTestWriter(t))
	log := zerolog.New(out).Level(zerolog.DebugLevel)
	ctx := log.WithContext(context.Background())
	client := createClient(ctx, t)
	r, err := client.TweetDetail(ctx, "1580661436132757506")
	if err != nil {
		t.Fatalf("TweetDetail returned error: %s", err)
	}
	if r.Tweet.Text != "a hit Tweet https://t.co/2C7cah4KzW" {
		t.Errorf("unexpected tweet text: %q", r.Tweet.Text)
	}
}
