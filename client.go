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
	ctx = log.WithContext(ctx)

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
				if ttw.TweetResults == nil || ttw.TweetResults.Result == nil {
					log.Debug().Msgf("missing tweet data in timeline tweet")
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
				r.Tweets = append(r.Tweets, tw.Tweet())
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
	for i, tw := range r.Tweets {
		c.backfillMissingReferencedTweets(ctx, &tw)
		r.Tweets[i] = tw
	}
	return r, nil
}

func errorFromResponse(resp *http.Response) error {
	if resp.StatusCode == 200 {
		return nil
	}
	if resp.StatusCode == 429 {
		return twitter.ErrThrottled
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

func (c *Client) tweetDetail(ctx context.Context, tweetID string) (*TweetDetailResponse, error) {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}

	log := zerolog.Ctx(ctx).With().
		Str("tweet_id", tweetID).
		Str("method", "TweetDetail").Logger()
	ctx = log.WithContext(ctx)

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
				if ttw.TweetResults == nil || ttw.TweetResults.Result == nil {
					log.Debug().Msgf("missing tweet data in timeline tweet")
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

				r.Tweet = tw.Tweet()
				r.RawJSON, _ = json.Marshal(ttw)
				return r, nil
			}
		}
	}

	return nil, fmt.Errorf("requested tweet is missing from the response")
}

func (c *Client) TweetDetail(ctx context.Context, tweetID string) (*TweetDetailResponse, error) {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}

	log := zerolog.Ctx(ctx).With().
		Str("tweet_id", tweetID).
		Str("method", "TweetDetail").Logger()
	ctx = log.WithContext(ctx)

	resp, err := c.tweetDetail(ctx, tweetID)
	if err != nil {
		return nil, err
	}
	c.backfillMissingReferencedTweets(ctx, &resp.Tweet)
	return resp, nil
}

func (c *Client) backfillMissingReferencedTweets(ctx context.Context, tw *twitter.Tweet) {
	log := zerolog.Ctx(ctx)
	refs := map[string]bool{}
	for _, r := range tw.ReferencedTweets {
		refs[r.ID] = true
	}
	for _, t := range tw.Includes.Tweets {
		delete(refs, t.ID)
	}
	for id := range refs {
		r, err := c.tweetDetail(ctx, id)
		if err != nil {
			log.Info().Err(err).Msgf("Failed to fetch tweet %q: %s", id, err)
		}
		// TODO(imax): merge in includes from r.Tweet
		tw.Includes.Tweets = append(tw.Includes.Tweets, r.Tweet.TweetNoIncludes)
	}
}

func (c *Client) UserTweetsAndReplies(ctx context.Context, userID string, cursor string) (*UserTweetsResponse, error) {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}

	log := zerolog.Ctx(ctx).With().
		Str("user_id", userID).
		Str("method", "UserTweetsAndReplies").Logger()
	ctx = log.WithContext(ctx)

	vars, features := userTweetsVarsAndFeatures(userID, cursor)
	params := url.Values{}
	params.Set("variables", vars)
	params.Set("features", features)

	req, err := http.NewRequestWithContext(ctx, "GET", graphQLQueryUrl("UserTweetsAndReplies")+"?"+params.Encode(), nil)
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
				if ttw.TweetResults == nil || ttw.TweetResults.Result == nil {
					log.Debug().Msgf("missing tweet data in timeline tweet")
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
				r.Tweets = append(r.Tweets, tw.Tweet())
			case *graphqlTimelineModule:
				for _, i := range c.Items {
					if i.Item.ItemContent == nil {
						continue
					}
					t, err := i.Item.ItemContent.Parse()
					if err != nil {
						log.Info().Err(err).Msgf("parsing item.itemContent")
						continue
					}
					tw, ok := t.(*graphqlTweet)
					if !ok {
						log.Info().Msgf("itemContent has unexpected type %T", tw)
						break
					}
					if tw.Legacy.AuthorID != userID {
						break
					}
					r.Tweets = append(r.Tweets, tw.Tweet())
				}
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
	for i, tw := range r.Tweets {
		c.backfillMissingReferencedTweets(ctx, &tw)
		r.Tweets[i] = tw
	}
	return r, nil
}

type UserByScreenNameResponse struct {
	RawJSON []byte
	ID      string
}

func (c *Client) UserByScreenName(ctx context.Context, username string) (*UserByScreenNameResponse, error) {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}

	log := zerolog.Ctx(ctx).With().
		Str("user_id", username).
		Str("method", "UserByScreenName").Logger()
	ctx = log.WithContext(ctx)

	vars, features := userByScreenNameVarsAndFeatures(username)
	params := url.Values{}
	params.Set("variables", vars)
	params.Set("features", features)

	req, err := http.NewRequestWithContext(ctx, "GET", graphQLQueryUrl("UserByScreenName")+"?"+params.Encode(), nil)
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

	data := &userByScreenNameResponse{}
	if err := json.NewDecoder(resp.Body).Decode(data); err != nil {
		return nil, fmt.Errorf("unmarshaling JSON response: %w", err)
	}

	r := &UserByScreenNameResponse{}
	r.RawJSON, _ = json.Marshal(data)

	v, err := data.Data.User.Result.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing data.user.result: %w", err)
	}

	u, ok := v.(*graphqlUser)
	if !ok {
		return nil, fmt.Errorf("data.user.result has unexpected type %q", data.Data.User.Result.TypeName)
	}

	r.ID = u.RestID

	return r, nil
}
