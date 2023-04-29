package pwitter

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/Ukraine-DAO/twitter-threads/twitter"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/rs/zerolog"

	. "github.com/rusni-pyzda/pwitter"
)

const testAccountID = "783214" // https://twitter.com/Twitter

var (
	client     *Client
	clientOnce sync.Once
)

func createClient(ctx context.Context, t *testing.T) *Client {
	clientOnce.Do(func() {
		auth, err := AnonymousAuth(ctx)
		if err != nil {
			t.Fatalf("Failed to construct authorizer: %s", err)
		}

		client = &Client{
			Authorizer: auth,
			Client:     http.DefaultClient,
		}
	})
	if client == nil {
		t.Fatalf("client was not constructed")
	}
	return client
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

func TestTweetContent(t *testing.T) {
	out := zerolog.NewConsoleWriter(zerolog.ConsoleTestWriter(t))
	log := zerolog.New(out).Level(zerolog.DebugLevel)
	ctx := log.WithContext(context.Background())
	client := createClient(ctx, t)

	cases := []struct {
		ID   string
		want string
	}{
		{
			ID:   "571542192939921408",
			want: `{"id":"571542192939921408","text":"RT @joncipriano: This is #LaunchHack. @TwitterDev @Launch @rchoi #hackathon http://t.co/hMRB11jObP","conversation_id":"571542192939921408","author_id":"2244994945","referenced_tweets":[{"type":"retweeted","id":"571540316437671937"}],"entities":{"urls":[{"start":76,"end":98,"url":"http://t.co/hMRB11jObP","expanded_url":"https://twitter.com/joncipriano/status/571540316437671937/video/1","display_url":"pic.twitter.com/hMRB11jObP"}],"hashtags":[{"start":25,"end":36,"tag":"LaunchHack"},{"start":65,"end":75,"tag":"hackathon"}],"mentions":[{"start":3,"end":15,"username":"joncipriano"},{"start":38,"end":49,"username":"TwitterDev"},{"start":50,"end":57,"username":"LAUNCH"},{"start":58,"end":64,"username":"rchoi"}]},"attachments":{"media_keys":["7_571540163135873024"]},"created_at":"2015-02-28T05:27:32.000Z","includes":{"users":[{"id":"2244994945","name":"Twitter Dev","username":"TwitterDev"},{"id":"4534871","name":"Jonathan Cipriano","username":"joncipriano"}],"media":[{"type":"video","media_key":"7_571540163135873024","preview_image_url":"https://pbs.twimg.com/ext_tw_video_thumb/571540163135873024/pu/img/aQFHH5pF_2BsvFql.jpg","variants":[{"bit_rate":832000,"content_type":"video/mp4","url":"https://video.twimg.com/ext_tw_video/571540163135873024/pu/vid/640x360/7iV9WnfpM_1UPEs4.mp4"},{"bit_rate":320000,"content_type":"video/mp4","url":"https://video.twimg.com/ext_tw_video/571540163135873024/pu/vid/320x180/RZ4aja3Jq7O9C80R.mp4"},{"bit_rate":2176000,"content_type":"video/mp4","url":"https://video.twimg.com/ext_tw_video/571540163135873024/pu/vid/1280x720/Xe02cv2UOdcCkeup.mp4"},{"content_type":"application/x-mpegURL","url":"https://video.twimg.com/ext_tw_video/571540163135873024/pu/pl/xR7iqWxLYqUurt2x.m3u8"}]}],"tweets":[{"id":"571540316437671937","text":"This is #LaunchHack. @TwitterDev @Launch @rchoi #hackathon http://t.co/hMRB11jObP","conversation_id":"571540316437671937","author_id":"4534871","entities":{"urls":[{"start":59,"end":81,"url":"http://t.co/hMRB11jObP","expanded_url":"https://twitter.com/joncipriano/status/571540316437671937/video/1","display_url":"pic.twitter.com/hMRB11jObP"}],"hashtags":[{"start":8,"end":19,"tag":"LaunchHack"},{"start":48,"end":58,"tag":"hackathon"}],"mentions":[{"start":21,"end":32,"username":"TwitterDev"},{"start":33,"end":40,"username":"LAUNCH"},{"start":41,"end":47,"username":"rchoi"}]},"attachments":{"media_keys":["7_571540163135873024"]},"created_at":"2015-02-28T05:20:04.000Z"}]}}`,
		},
		{
			ID:   "560273169542443008",
			want: `{"id":"560273169542443008","text":"MC @noonisms kicks off our meetup tonight in Las Vegas at @Zappos - #TwitterDrive to join the conversation http://t.co/NbKpDEXDB3","conversation_id":"560273169542443008","author_id":"2244994945","entities":{"urls":[{"start":107,"end":129,"url":"http://t.co/NbKpDEXDB3","expanded_url":"https://twitter.com/TwitterDev/status/560273169542443008/photo/1","display_url":"pic.twitter.com/NbKpDEXDB3"}],"hashtags":[{"start":68,"end":81,"tag":"TwitterDrive"}],"mentions":[{"start":3,"end":12,"username":"noonisms"},{"start":58,"end":65,"username":"Zappos"}]},"attachments":{"media_keys":["3_560273169164931072"]},"created_at":"2015-01-28T03:08:27.000Z","includes":{"users":[{"id":"2244994945","name":"Twitter Dev","username":"TwitterDev"}],"media":[{"type":"photo","media_key":"3_560273169164931072","url":"https://pbs.twimg.com/media/B8Z9eplCEAA5Ewp.png"}]}}`,
		},
	}

	for _, tc := range cases {
		want := &twitter.Tweet{}
		json.Unmarshal([]byte(tc.want), want)

		r, err := client.TweetDetail(ctx, tc.ID)
		if err != nil {
			t.Errorf("TweetDetail returned error: %s", err)
			continue
		}

		// We might return extra entries in includes.users and that's ok.
		wantUser := map[string]bool{}
		for _, u := range want.Includes.Users {
			wantUser[u.ID] = true
		}
		filteredUsers := []twitter.TwitterUser{}
		for _, u := range r.Tweet.Includes.Users {
			if wantUser[u.ID] {
				filteredUsers = append(filteredUsers, u)
			}
		}
		r.Tweet.Includes.Users = filteredUsers

		sortVideoVariants := cmpopts.SortSlices(func(a map[string]interface{}, b map[string]interface{}) bool {
			aa, oka := a["bit_rate"].(float64)
			bb, okb := b["bit_rate"].(float64)
			if oka && okb {
				return aa < bb
			}
			return okb
		})

		diff := cmp.Diff(want, &r.Tweet, sortVideoVariants)
		if diff != "" {
			t.Errorf("%s", diff)
		}
	}

}
