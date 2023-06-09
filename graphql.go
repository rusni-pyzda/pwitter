package pwitter

import (
	"encoding/json"
	"fmt"
)

var (
	graphqlID = map[string]string{
		"UserTweets":           "HuTx74BxAnezK1gWvYY7zg",
		"TweetDetail":          "BbCrSoXIR7z93lLCVFlQ2Q",
		"UserByRestId":         "GazOglcBvgLigl3ywt6b3Q",
		"UserTweetsAndReplies": "zQxfEr5IFxQ2QZ-XMJlKew",
		"UserByScreenName":     "sLVLhk0bGj3MVFEKTdax1w",
	}
)

const (
	// TODO: extract from html on the fly
	twitterFeatures = `{"blue_business_profile_image_shape_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"tweetypie_unmention_optimization_enabled":true,"vibe_api_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":false,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":false,"interactive_text_enabled":true,"responsive_web_text_conversations_enabled":false,"longform_notetweets_rich_text_read_enabled":true,"responsive_web_enhance_cards_enabled":false}`
)

func graphQLQueryUrl(queryName string) string {
	return fmt.Sprintf("https://twitter.com/i/api/graphql/%s/%s", graphqlID[queryName], queryName)
}

type userTweetsVariables struct {
	UserID                                 string `json:"userId,omitempty"`
	Count                                  int    `json:"count"`
	IncludePromotedContent                 bool   `json:"includePromotedContent"`
	WithQuickPromoteEligibilityTweetFields bool   `json:"withQuickPromoteEligibilityTweetFields"`
	WithVoice                              bool   `json:"withVoice"`
	WithV2Timeline                         bool   `json:"withV2Timeline"`
	Cursor                                 string `json:"cursor,omitempty"`
}

func userTweetsVarsAndFeatures(userID string, cursor string) (string, string) {
	v := &userTweetsVariables{
		UserID:                                 userID,
		Count:                                  40,
		IncludePromotedContent:                 true,
		WithQuickPromoteEligibilityTweetFields: true,
		WithVoice:                              true,
		WithV2Timeline:                         true,
		Cursor:                                 cursor,
	}

	vars, _ := json.Marshal(v)
	return string(vars), twitterFeatures
}

type errors []json.RawMessage

type userTweetsResponse struct {
	Data struct {
		User struct {
			Result *graphqlObject `json:"result"`
		} `json:"user"`
	} `json:"data"`
	Errors errors `json:"errors,omitempty"`
}

type tweetDetailVariables struct {
	ID                                     string `json:"focalTweetId"`
	WithRuxInjections                      bool   `json:"with_rux_injections"`
	IncludePromotedContent                 bool   `json:"includePromotedContent"`
	WithQuickPromoteEligibilityTweetFields bool   `json:"withQuickPromoteEligibilityTweetFields"`
	WithVoice                              bool   `json:"withVoice"`
	WithV2Timeline                         bool   `json:"withV2Timeline"`
	WithCommunity                          bool   `json:"withCommunity"`
	WithBirdwatchNotes                     bool   `json:"withBirdwatchNotes"`
}

func tweetDetailVarsAndFeatures(id string) (string, string) {
	v := &tweetDetailVariables{
		ID:                                     id,
		IncludePromotedContent:                 true,
		WithQuickPromoteEligibilityTweetFields: true,
		WithVoice:                              true,
		WithV2Timeline:                         true,
		WithCommunity:                          true,
	}

	vars, _ := json.Marshal(v)
	return string(vars), twitterFeatures
}

type tweetDetailResponse struct {
	Data struct {
		ThreadedConversationWithInjectionsV2 struct {
			Instructions []timelineInstruction `json:"instructions"`
		} `json:"threaded_conversation_with_injections_v2"`
	} `json:"data"`
	Errors errors `json:"errors,omitempty"`
}

type userByScreenNameVariables struct {
	ScreenName               string `json:"screen_name"`
	WithSafetyModeUserFields bool   `json:"withSafetyModeUserFields"`
}

func userByScreenNameVarsAndFeatures(username string) (string, string) {
	v := &userByScreenNameVariables{
		ScreenName:               username,
		WithSafetyModeUserFields: true,
	}

	vars, _ := json.Marshal(v)
	return string(vars), twitterFeatures
}

type userByScreenNameResponse struct {
	Data struct {
		User struct {
			Result *graphqlObject `json:"result"`
		} `json:"user"`
	} `json:"data"`
	Errors errors `json:"errors,omitempty"`
}
