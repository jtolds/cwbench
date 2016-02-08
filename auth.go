// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"encoding/json"
	"flag"
	"net/http"

	"github.com/jtolds/webhelp"
	oauth2p "github.com/jtolds/webhelp-oauth2"
	"golang.org/x/net/context"
)

var (
	googleClientId     = flag.String("google_client_id", "", "")
	googleClientSecret = flag.String("google_client_secret", "", "")
	visibleURL         = flag.String("visible_url", "http://localhost:8080", "")

	oauth2 *oauth2p.ProviderHandler
)

func loadOAuth2() {
	oauth2 = oauth2p.NewProviderHandler(
		oauth2p.Google(oauth2p.Config{
			ClientID:     *googleClientId,
			ClientSecret: *googleClientSecret,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
			RedirectURL:  *visibleURL + "/auth/_cb"}),
		"oauth-google", "/auth",
		oauth2p.RedirectURLs{})
}

type UserInfo struct {
	Id            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Link          string `json:"link"`
	Picture       string `json:"picture"`
}

func (a *Endpoints) LoadUser(ctx context.Context, inr *http.Request) (
	*UserInfo, error) {
	if inr.FormValue("api_key") != "" {
		key, err := a.Data.APIKey(inr.FormValue("api_key"))
		if err != nil {
			return nil, err
		}
		if key == nil {
			return nil, webhelp.ErrUnauthorized.New("invalid api key")
		}
		return &UserInfo{Id: key.UserId}, nil
	}

	t, err := oauth2.Token(ctx)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, nil
	}

	outr, err := http.NewRequest("GET",
		"https://www.googleapis.com/oauth2/v1/userinfo", nil)
	if err != nil {
		return nil, err
	}
	t.SetAuthHeader(outr)
	resp, err := http.DefaultClient.Do(outr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, webhelp.HTTPError.New("invalid status: %v", resp.Status)
	}
	var data UserInfo
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

type loginRequired struct {
	a *Endpoints
	h webhelp.Handler
}

func (a *Endpoints) LoginRequired(h webhelp.Handler) webhelp.Handler {
	return loginRequired{a: a, h: h}
}

type ctxKey int

var (
	userKey ctxKey = 1
)

func (lr loginRequired) HandleHTTP(ctx context.Context,
	w webhelp.ResponseWriter, r *http.Request) error {
	user, err := lr.a.LoadUser(ctx, r)
	if err != nil {
		return err
	}
	if user == nil {
		return webhelp.Redirect(w, r, oauth2.LoginURL(r.RequestURI, false))
	}
	return lr.h.HandleHTTP(context.WithValue(ctx, userKey, user), w, r)
}

func (lr loginRequired) Routes(
	cb func(method, path string, annotations []string)) {
	webhelp.Routes(lr.h, cb)
}

var _ webhelp.Handler = loginRequired{}
var _ webhelp.RouteLister = loginRequired{}

func LoadUser(ctx context.Context) *UserInfo {
	return ctx.Value(userKey).(*UserInfo)
}
