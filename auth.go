package pwitter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

type Authorizer interface {
	SetAuthHeader(req *http.Request)
}

type anonAuthInfo struct {
	BearerToken string
	GuestID     string
	GuestToken  string
	CSRFToken   string
}

func getAnonAuthInfo(ctx context.Context) (*anonAuthInfo, error) {
	log := zerolog.Ctx(ctx)

	r := &anonAuthInfo{}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("creating a cookie jar: %w", err)
	}
	client := &http.Client{Jar: jar}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://twitter.com", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request object: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching html page: %w", err)
	}
	defer resp.Body.Close()
	log.Trace().Msgf("Response headers: %+v", resp.Header)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server returned unexpected response: %s\n%+v", resp.Status, resp)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading html response: %w", err)
	}
	matches := regexp.MustCompile(`https://[^"]+/main.[^.]+.js`).FindAll(b, 1)
	if len(matches) < 1 {
		return nil, fmt.Errorf("didn't find a URL of the main js file")
	}

	u, _ := url.Parse("https://twitter.com")
	for _, c := range jar.Cookies(u) {
		switch c.Name {
		case "guest_id":
			r.GuestID = c.Value
		case "gt":
			r.GuestToken = c.Value
		}
	}
	if r.GuestToken == "" {
		matches := regexp.MustCompile(`document.cookie="gt=([^;"]+);`).FindAllSubmatch(b, 1)
		if len(matches) >= 1 {
			r.GuestToken = string(matches[0][1])
		}
	}

	if r.GuestID == "" || r.GuestToken == "" {
		return nil, fmt.Errorf("guest_id/gt must be not empty (%q/%q)", r.GuestID, r.GuestToken)
	}

	req, err = http.NewRequestWithContext(ctx, "GET", string(matches[0]), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request object: %w", err)
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching js: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server returned unexpected response: %s\n%+v", resp.Status, resp)
	}
	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading js response: %w", err)
	}
	matches = regexp.MustCompile(`AAAAAAAAA[^"]+`).FindAll(b, 1)
	if len(matches) < 1 {
		return nil, fmt.Errorf("didn't find the token in main js file")
	}

	r.BearerToken = string(matches[0])

	b = make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	r.CSRFToken = hex.EncodeToString(b)

	return r, nil
}

type AnonymousAuthorizer struct {
	info anonAuthInfo
}

// TODO: MarshalJSON/UnmarshalJSON

func AnonymousAuth(ctx context.Context) (*AnonymousAuthorizer, error) {
	i, err := getAnonAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &AnonymousAuthorizer{info: *i}, nil
}

func (a *AnonymousAuthorizer) SetAuthHeader(req *http.Request) {
	cookie := func(name string, value string) string {
		return (&http.Cookie{Name: name, Value: value}).String()
	}
	cookies := []string{
		cookie("guest_id", a.info.GuestID),
		cookie("gt", a.info.GuestToken),
		cookie("ct0", a.info.CSRFToken),
		cookie("dnt", "1"),
	}
	req.Header.Set("Cookie", strings.Join(cookies, "; "))
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", a.info.BearerToken))
	req.Header.Set("x-csrf-token", a.info.CSRFToken)
	req.Header.Set("x-guest-token", a.info.GuestToken)
	req.Header.Set("DNT", "1")
}
