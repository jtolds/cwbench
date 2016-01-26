// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"encoding/json"
	"net/http"

	"github.com/jtolds/webhelp"
	"golang.org/x/net/context"
)

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

func LoadUser(ctx context.Context) (*UserInfo, error) {
	t, err := oauth2.Token(ctx)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, nil
	}

	r, err := http.NewRequest("GET",
		"https://www.googleapis.com/oauth2/v1/userinfo", nil)
	if err != nil {
		return nil, err
	}
	t.SetAuthHeader(r)
	resp, err := http.DefaultClient.Do(r)
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
