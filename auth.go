// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"encoding/json"
	"flag"
	"net/http"

	"golang.org/x/net/context"
	"gopkg.in/go-webhelp/whoauth2.v1"
	"gopkg.in/webhelp.v1/whcompat"
	"gopkg.in/webhelp.v1/wherr"
	"gopkg.in/webhelp.v1/whredir"
	"gopkg.in/webhelp.v1/whroute"
)

var (
	googleClientId     = flag.String("google_client_id", "", "")
	googleClientSecret = flag.String("google_client_secret", "", "")
	visibleURL         = flag.String("visible_url", "http://localhost:8080", "")

	oauth2 *whoauth2.ProviderHandler
)

func loadOAuth2() {
	oauth2 = whoauth2.NewProviderHandler(
		whoauth2.Google(whoauth2.Config{
			ClientID:     *googleClientId,
			ClientSecret: *googleClientSecret,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
			RedirectURL:  *visibleURL + "/auth/_cb"}),
		"oauth-google", "/auth",
		whoauth2.RedirectURLs{})
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
			return nil, wherr.Unauthorized.New("invalid api key")
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
		return nil, wherr.HTTPError.New("invalid status: %v", resp.Status)
	}
	var data UserInfo
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

type ctxKey int

var (
	userKey ctxKey = 1
)

func (a *Endpoints) LoginRequired(h http.Handler) http.Handler {
	return whroute.HandlerFunc(h,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := whcompat.Context(r)
			user, err := a.LoadUser(ctx, r)
			if err != nil {
				wherr.Handle(w, r, err)
				return
			}
			if user == nil {
				whredir.Redirect(w, r, oauth2.LoginURL(r.RequestURI, false))
				return
			}
			ctx = context.WithValue(ctx, userKey, user)
			h.ServeHTTP(w, whcompat.WithContext(r, ctx))
		})
}

func LoadUser(ctx context.Context) *UserInfo {
	return ctx.Value(userKey).(*UserInfo)
}
