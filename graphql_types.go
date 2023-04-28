package pwitter

import (
	"encoding/json"
	"fmt"

	"github.com/Ukraine-DAO/twitter-threads/twitter"
)

var (
	graphqlType = map[string]func() interface{}{
		"TimelineTimelineCursor": func() interface{} { return &graphqlTimelineCursor{} },
		"TimelineTimelineItem":   func() interface{} { return &graphqlTimelineItem{} },
		"TimelineTweet":          func() interface{} { return &graphqlTimelineTweet{} },
		"Tweet":                  func() interface{} { return &graphqlTweet{} },
		"User":                   func() interface{} { return &graphqlUser{} },
	}
)

type graphqlObject struct {
	TypeName string `json:"__typename"`
	RawJSON  []byte
}

type parsedGraphqlObject interface{}

func (o *graphqlObject) UnmarshalJSON(b []byte) error {
	v := struct {
		TypeName string `json:"__typename"`
	}{}
	o.RawJSON = make([]byte, len(b))
	copy(o.RawJSON, b)
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	o.TypeName = v.TypeName
	return nil
}

func (o *graphqlObject) MarshalJSON() ([]byte, error) {
	if len(o.RawJSON) == 0 {
		return []byte("null"), nil
	}
	return o.RawJSON, nil
}

func (o *graphqlObject) Parse() (parsedGraphqlObject, error) {
	if o.TypeName == "" {
		return nil, fmt.Errorf("object doesn't have a type annotation")
	}
	mk := graphqlType[o.TypeName]
	if mk == nil {
		return nil, fmt.Errorf("handler for type %q is not implemented", o.TypeName)
	}
	r := mk()
	if r == nil {
		return nil, fmt.Errorf("handler for type %q returned nil", o.TypeName)
	}
	if err := json.Unmarshal(o.RawJSON, r); err != nil {
		return nil, fmt.Errorf("unmarshaling JSON: %w", err)
	}
	return r, nil
}

type graphqlUser struct {
	ID         string `json:"id,omitempty"`
	RestID     string `json:"rest_id,omitempty"`
	TimelineV2 *struct {
		Timeline struct {
			Instructions []timelineInstruction `json:"instructions"`
		} `json:"timeline"`
	} `json:"timeline_v2,omitempty"`
	Legacy *graphqlUserLegacy `json:"legacy,omitempty"`
}

type graphqlUserLegacy struct {
	Name       string `json:"name"`
	ScreenName string `json:"screen_name"`
}

type timelineInstruction struct {
	Type    timelineInstructionType    `json:"type"`
	Entries []timelineInstructionEntry `json:"entries,omitempty"`
	Entry   *timelineInstructionEntry  `json:"entry,omitempty"`
}

type timelineInstructionType string

const (
	timelineAddEntries timelineInstructionType = "TimelineAddEntries"
	timelineClearCache                         = "TimelineClearCache"
	timelinePinEntry                           = "TimelinePinEntry"
)

type timelineInstructionEntry struct {
	EntryID   string         `json:"entryId,omitempty"`
	SortIndex string         `json:"sortIndex,omitempty"`
	Content   *graphqlObject `json:"content,omitempty"`
}

type graphqlTimelineItem struct {
	EntryType   string         `json:"entryType,omitempty"`
	ItemContent *graphqlObject `json:"itemContent,omitempty"`
}

type graphqlTimelineTweet struct {
	ItemType     string               `json:"itemType,omitempty"`
	TweetResults *graphqlTweetResults `json:"tweet_results,omitempty"`
	DisplayType  string               `json:"tweetDisplayType,omitempty"`
}

type graphqlTweetResults struct {
	Result *graphqlObject `json:"result,omitempty"`
}

type graphqlTweet struct {
	RestID string             `json:"rest_id,omitempty"`
	Legacy graphqlTweetLegacy `json:"legacy,omitempty"`
	Source string             `json:"source,omitempty"`
	Core   graphqlTweetCore   `json:"core,omitempty"`
}

type graphqlTweetCore struct {
	UserResults struct {
		Result *graphqlObject `json:"result,omitempty"`
	} `json:"user_results,omitempty"`
}

func (t *graphqlTweet) Tweet() twitter.Tweet {
	r := twitter.Tweet{
		TweetNoIncludes: twitter.TweetNoIncludes{
			ID:              t.RestID,
			Text:            t.Legacy.Text,
			AuthorID:        t.Legacy.AuthorID,
			ConversationID:  t.Legacy.ConversationID,
			CreatedAt:       t.Legacy.CreatedAt,
			InReplyToUserID: t.Legacy.InReplyToUserID,
		},
	}
	if t.Core.UserResults.Result != nil {
		u, err := t.Core.UserResults.Result.Parse()
		if err == nil {
			u, ok := u.(*graphqlUser)
			if ok {
				if u.Legacy != nil {
					r.Includes.Users = append(r.Includes.Users, twitter.TwitterUser{
						ID:       u.RestID,
						Name:     u.Legacy.Name,
						Username: u.Legacy.ScreenName,
					})
				}
			}
		}
	}
	if replyTo := t.Legacy.InReplyToStatusID; replyTo != "" {
		r.ReferencedTweets = append(r.ReferencedTweets,
			twitter.ReferencedTweet{Type: "replied_to", ID: replyTo})
	}
	if quoted := t.Legacy.QuotedStatusID; quoted != "" {
		r.ReferencedTweets = append(r.ReferencedTweets,
			twitter.ReferencedTweet{Type: "quoted", ID: quoted})
	}
	if rt := t.Legacy.RetweetedStatusResult; rt != nil {
		if rt.Result != nil {
			tw, err := rt.Result.Parse()
			if err == nil {
				tw, ok := tw.(*graphqlTweet)
				if ok {
					r.ReferencedTweets = append(r.ReferencedTweets,
						twitter.ReferencedTweet{Type: "retweeted", ID: tw.RestID})
					converted := tw.Tweet()
					r.Includes.Tweets = append(r.Includes.Tweets, converted.TweetNoIncludes)
					r.Includes.Media = append(r.Includes.Media, converted.Includes.Media...)
					r.Includes.Users = append(r.Includes.Users, converted.Includes.Users...)
					r.Includes.Tweets = append(r.Includes.Tweets, converted.Includes.Tweets...)
				}
			}
		}
	}

	if t.Legacy.Entities != nil {
		for _, ue := range t.Legacy.Entities.URLs {
			r.Entities.URLs = append(r.Entities.URLs, twitter.EntityURL{
				TextEntity:  twitter.TextEntity{Start: uint16(ue.Indices[0]), End: uint16(ue.Indices[1])},
				URL:         ue.URL,
				ExpandedURL: ue.ExpandedURL,
				DisplayURL:  ue.DisplayURL,
			})
		}
		for _, he := range t.Legacy.Entities.Hashtags {
			r.Entities.Hashtags = append(r.Entities.Hashtags, twitter.EntityHashtag{
				TextEntity: twitter.TextEntity{Start: uint16(he.Indices[0]), End: uint16(he.Indices[1])},
				Tag:        he.Text,
			})
		}
		for _, me := range t.Legacy.Entities.UserMentions {
			r.Entities.Mentions = append(r.Entities.Mentions, twitter.EntityMention{
				TextEntity: twitter.TextEntity{Start: uint16(me.Indices[0]), End: uint16(me.Indices[1])},
				Username:   me.ScreenName,
			})
			r.Includes.Users = append(r.Includes.Users, twitter.TwitterUser{
				ID:       me.ID,
				Name:     me.Name,
				Username: me.ScreenName,
			})
		}
	}

	if t.Legacy.ExtendedEntities != nil {
		for _, e := range t.Legacy.ExtendedEntities.Media {
			r.Attachments.MediaKeys = append(r.Attachments.MediaKeys, e.MediaKey)
			r.Entities.URLs = append(r.Entities.URLs, twitter.EntityURL{
				TextEntity:  twitter.TextEntity{Start: uint16(e.Indices[0]), End: uint16(e.Indices[1])},
				URL:         e.URL,
				ExpandedURL: e.ExpandedURL,
				DisplayURL:  e.DisplayURL,
			})
			m := twitter.Media{
				Type:       e.Type,
				Key:        e.MediaKey,
				URL:        e.ExpandedURL,
				PreviewURL: e.MediaURLHTTPS,
			}
			if e.VideoInfo != nil {
				_ = json.Unmarshal(e.VideoInfo.Variants, &m.Variants)
			}
			r.Includes.Media = append(r.Includes.Media, m)
		}
	}

	// Includes.Tweets

	return r
}

type graphqlTweetLegacy struct {
	ID                    string    `json:"id_str,omitempty"`
	CreatedAt             string    `json:"created_at,omitempty"`
	ConversationID        string    `json:"conversation_id_str,omitempty"`
	Text                  string    `json:"full_text,omitempty"`
	AuthorID              string    `json:"user_id_str,omitempty"`
	Likes                 int       `json:"favorite_count,omitempty"`
	Replies               int       `json:"reply_count,omitempty"`
	Retweets              int       `json:"retweet_count,omitempty"`
	Quotes                int       `json:"quote_count,omitempty"`
	InReplyToUserID       string    `json:"in_reply_to_user_id_str"`
	InReplyToStatusID     string    `json:"in_reply_to_status_id_str"`
	QuotedStatusID        string    `json:"quoted_status_id_str"`
	Entities              *entities `json:"entities,omitempty"`
	ExtendedEntities      *entities `json:"extended_entities,omitempty"`
	RetweetedStatusResult *struct {
		Result *graphqlObject `json:"result"`
	} `json:"retweeted_status_result"`
}

type graphqlTimelineCursor struct {
	EntryType           string `json:"entryType,omitempty"`
	Value               string `json:"value,omitempty"`
	CursorType          string `json:"cursorType,omitempty"`
	StopOnEmptyResponse bool   `json:"stopOnEmptyResponse,omitempty"`
}

type entities struct {
	Media        []entityMedia       `json:"media"`
	URLs         []entityURL         `json:"urls"`
	UserMentions []entityUserMention `json:"user_mentions"`
	Hashtags     []entityHashtag     `json:"hashtags"`
	//Symbols      []entitySymbol      `json:"symbols"`
}

type entityMedia struct {
	Type                 string `json:"type"`
	ID                   string `json:"id_str"`
	URL                  string `json:"url"`
	DisplayURL           string `json:"display_url"`
	ExpandedURL          string `json:"expanded_url"`
	Indices              [2]int `json:"indices"`
	MediaURLHTTPS        string `json:"media_url_https"`
	MediaKey             string `json:"media_key"`
	ExtMediaAvailability struct {
		Status string `json:"status"`
	} `json:"ext_media_availability"`
	Sizes        struct{}   `json:"sizes"`
	OriginalInfo struct{}   `json:"original_info"`
	VideoInfo    *videoInfo `json:"video_info"`
}

type videoInfo struct {
	Variants json.RawMessage `json:"variants"`
}

type entityURL struct {
	DisplayURL  string `json:"display_url"`
	ExpandedURL string `json:"expanded_url"`
	URL         string `json:"url"`
	Indices     [2]int `json:"indices"`
}

type entityHashtag struct {
	Text    string `json:"text"`
	Indices [2]int `json:"indices"`
}

type entityUserMention struct {
	Indices    [2]int `json:"indices"`
	Name       string `json:"name"`
	ScreenName string `json:"screen_name"`
	ID         string `json:"id_str"`
}
