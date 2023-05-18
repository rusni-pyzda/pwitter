package tweetdiff

import (
	"github.com/Ukraine-DAO/twitter-threads/twitter"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func Diff(public *twitter.Tweet, private *twitter.Tweet) string {
	public.RequestConfig = private.RequestConfig

	// We might return extra entries in includes.users and that's ok.
	wantUser := map[string]bool{}
	for _, u := range public.Includes.Users {
		wantUser[u.ID] = true
	}
	filteredUsers := []twitter.TwitterUser{}
	for _, u := range private.Includes.Users {
		if wantUser[u.ID] {
			filteredUsers = append(filteredUsers, u)
		}
	}
	private.Includes.Users = filteredUsers

	// Same with media
	wantMedia := map[string]bool{}
	for _, u := range public.Includes.Media {
		wantMedia[u.Key] = true
	}
	filteredMedia := []twitter.Media{}
	for _, u := range private.Includes.Media {
		if wantMedia[u.Key] {
			filteredMedia = append(filteredMedia, u)
		}
	}
	private.Includes.Media = filteredMedia

	sortVideoVariants := cmpopts.SortSlices(func(a map[string]interface{}, b map[string]interface{}) bool {
		aa, oka := a["bit_rate"].(float64)
		bb, okb := b["bit_rate"].(float64)
		if oka && okb {
			return aa < bb
		}
		return okb
	})

	return cmp.Diff(public, private, sortVideoVariants, cmpopts.EquateEmpty())
}
