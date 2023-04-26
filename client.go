package pwitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Ukraine-DAO/twitter-threads/twitter"
	"github.com/rs/zerolog"
)

type Client struct {
	Authorizer Authorizer
	Client     *http.Client
}

type UserTweetsResponse struct {
	RawJSON    []byte
	Tweets     []twitter.Tweet
	CursorNext string
	CursorPrev string
}

func (c *Client) UserTweets(ctx context.Context, userID string, cursor string) (*UserTweetsResponse, error) {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}

	log := zerolog.Ctx(ctx).With().
		Str("user_id", userID).
		Str("method", "UserTweets").Logger()

	vars, features := userTweetsVarsAndFeatures(userID, cursor)
	params := url.Values{}
	params.Set("variables", vars)
	params.Set("features", features)

	req, err := http.NewRequestWithContext(ctx, "GET", graphQLQueryUrl("UserTweets")+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request object: %w", err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")

	c.Authorizer.SetAuthHeader(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if err := errorFromResponse(resp); err != nil {
		return nil, fmt.Errorf("UserTweets: %w", err)
	}

	data := &userTweetsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(data); err != nil {
		return nil, fmt.Errorf("unmarshaling JSON response: %w", err)
	}

	r := &UserTweetsResponse{}
	r.RawJSON, _ = json.Marshal(data)

	v, err := data.Data.User.Result.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing data.user.result: %w", err)
	}

	u, ok := v.(*graphqlUser)
	if !ok {
		return nil, fmt.Errorf("data.user.result has unexpected type %q", data.Data.User.Result.TypeName)
	}

	timeline := u.TimelineV2
	if timeline == nil {
		return nil, fmt.Errorf("no timeline found in the response")
	}

	for _, instr := range timeline.Timeline.Instructions {
		if instr.Type != timelineAddEntries {
			continue
		}
		for _, e := range instr.Entries {
			c, err := e.Content.Parse()
			if err != nil {
				log.Info().Msgf("failed to parse instruction content: %s", err)
				continue
			}
			switch c := c.(type) {
			case *graphqlTimelineItem:
				if c.ItemContent.TypeName != "TimelineTweet" {
					break
				}
				t, err := c.ItemContent.Parse()
				if err != nil {
					log.Info().Msgf("failed to parse item content: %s", err)
					break
				}
				ttw, ok := t.(*graphqlTimelineTweet)
				if !ok {
					log.Info().Msgf("item content has unexpected type %T", ttw)
					break
				}
				t, err = ttw.TweetResults.Result.Parse()
				if err != nil {
					log.Info().Msgf("failed to parse tweet results: %s", err)
					break
				}
				tw, ok := t.(*graphqlTweet)
				if !ok {
					log.Info().Msgf("tweet results have unexpected type %T", tw)
					break
				}
				if tw.Legacy.AuthorID != userID {
					break
				}
				// TODO: entities
				tweet := twitter.Tweet{
					TweetNoIncludes: twitter.TweetNoIncludes{
						ID:              tw.RestID,
						Text:            tw.Legacy.Text,
						AuthorID:        tw.Legacy.AuthorID,
						ConversationID:  tw.Legacy.ConversationID,
						CreatedAt:       tw.Legacy.CreatedAt,
						InReplyToUserID: tw.Legacy.InReplyToUserID,
						//ReferencedTweets: nil,
						//Entities:
						//Attachments:
					},
					//Includes:,
				}
				if replyTo := tw.Legacy.InReplyToStatusID; replyTo != "" {
					tweet.ReferencedTweets = append(tweet.ReferencedTweets,
						twitter.ReferencedTweet{Type: "replied_to", ID: replyTo})
				}
				// quoted
				// retweeted

				r.Tweets = append(r.Tweets, tweet)
			case *graphqlTimelineCursor:
				switch c.CursorType {
				case "Top":
					r.CursorPrev = c.Value
				case "Bottom":
					r.CursorNext = c.Value
				}
			}
		}
	}
	return r, nil
}

func errorFromResponse(resp *http.Response) error {
	if resp.StatusCode == 200 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	headers := []string{}
	for k, vs := range resp.Header {
		for _, v := range vs {
			headers = append(headers, fmt.Sprintf("%s: %s", k, v))
		}
	}
	return fmt.Errorf("got error response: %d %s\n%s\n\n%s", resp.StatusCode, resp.Status, strings.Join(headers, "\n"), string(body))
}

type TweetDetailResponse struct {
	RawJSON []byte
	Tweet   twitter.Tweet
}

func (c *Client) TweetDetail(ctx context.Context, tweetID string) (*TweetDetailResponse, error) {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}

	log := zerolog.Ctx(ctx).With().
		Str("tweet_id", tweetID).
		Str("method", "TweetDetail").Logger()

	vars, features := tweetDetailVarsAndFeatures(tweetID)
	params := url.Values{}
	params.Set("variables", vars)
	params.Set("features", features)

	req, err := http.NewRequestWithContext(ctx, "GET", graphQLQueryUrl("TweetDetail")+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request object: %w", err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")

	c.Authorizer.SetAuthHeader(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if err := errorFromResponse(resp); err != nil {
		return nil, fmt.Errorf("UserTweets: %w", err)
	}

	data := &tweetDetailResponse{}
	if err := json.NewDecoder(resp.Body).Decode(data); err != nil {
		return nil, fmt.Errorf("unmarshaling JSON response: %w", err)
	}

	r := &TweetDetailResponse{}

	for _, instr := range data.Data.ThreadedConversationWithInjectionsV2.Instructions {
		if instr.Type != timelineAddEntries {
			continue
		}

		for _, e := range instr.Entries {
			c, err := e.Content.Parse()
			if err != nil {
				log.Info().Msgf("failed to parse instruction content: %s", err)
				continue
			}
			switch c := c.(type) {
			case *graphqlTimelineItem:
				if c.ItemContent.TypeName != "TimelineTweet" {
					break
				}
				t, err := c.ItemContent.Parse()
				if err != nil {
					log.Info().Msgf("failed to parse item content: %s", err)
					break
				}
				ttw, ok := t.(*graphqlTimelineTweet)
				if !ok {
					log.Info().Msgf("item content has unexpected type %T", ttw)
					break
				}
				t, err = ttw.TweetResults.Result.Parse()
				if err != nil {
					log.Info().Msgf("failed to parse tweet results: %s", err)
					break
				}
				tw, ok := t.(*graphqlTweet)
				if !ok {
					log.Info().Msgf("tweet results have unexpected type %T", tw)
					break
				}
				if tw.Legacy.ID != tweetID {
					break
				}
				// TODO: entities
				tweet := twitter.Tweet{
					TweetNoIncludes: twitter.TweetNoIncludes{
						ID:              tw.RestID,
						Text:            tw.Legacy.Text,
						AuthorID:        tw.Legacy.AuthorID,
						ConversationID:  tw.Legacy.ConversationID,
						CreatedAt:       tw.Legacy.CreatedAt,
						InReplyToUserID: tw.Legacy.InReplyToUserID,
						//ReferencedTweets: nil,
						//Entities:
						//Attachments:
					},
					//Includes:,
				}
				if replyTo := tw.Legacy.InReplyToStatusID; replyTo != "" {
					tweet.ReferencedTweets = append(tweet.ReferencedTweets,
						twitter.ReferencedTweet{Type: "replied_to", ID: replyTo})
				}
				// quoted
				// retweeted

				r.Tweet = tweet
				r.RawJSON, _ = json.Marshal(ttw)
				return r, nil
			}
		}
	}

	return nil, fmt.Errorf("requested tweet is missing from the response")
}
