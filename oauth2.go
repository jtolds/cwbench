// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"flag"

	oauth2p "github.com/jtolds/webhelp-oauth2"
)

var (
	googleClientId     = flag.String("google_client_id", "", "")
	googleClientSecret = flag.String("google_client_secret", "", "")

	oauth2 *oauth2p.ProviderHandler
)

func loadOAuth2() {
	oauth2 = oauth2p.NewProviderHandler(
		oauth2p.Google(oauth2p.Config{
			ClientID:     *googleClientId,
			ClientSecret: *googleClientSecret,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
			RedirectURL:  "http://localhost:8080/auth/_cb"}),
		"oauth-google", "/auth",
		oauth2p.RedirectURLs{})
}
