package pwitter

import (
	"encoding/json"
	"fmt"
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
}

type graphqlTweetLegacy struct {
	ID                string    `json:"id_str,omitempty"`
	CreatedAt         string    `json:"created_at,omitempty"`
	ConversationID    string    `json:"conversation_id_str,omitempty"`
	Text              string    `json:"full_text,omitempty"`
	AuthorID          string    `json:"user_id_str,omitempty"`
	Likes             int       `json:"favorite_count,omitempty"`
	Replies           int       `json:"reply_count,omitempty"`
	Retweets          int       `json:"retweet_count,omitempty"`
	Quotes            int       `json:"quote_count,omitempty"`
	InReplyToUserID   string    `json:"in_reply_to_user_id_str"`
	InReplyToStatusID string    `json:"in_reply_to_status_id_str"`
	Entities          *entities `json:"entities,omitempty"`
	ExtendedEntities  *entities `json:"extended_entities,omitempty"`
}

type graphqlTimelineCursor struct {
	EntryType           string `json:"entryType,omitempty"`
	Value               string `json:"value,omitempty"`
	CursorType          string `json:"cursorType,omitempty"`
	StopOnEmptyResponse bool   `json:"stopOnEmptyResponse,omitempty"`
}
