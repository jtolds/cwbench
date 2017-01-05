// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"gopkg.in/webhelp.v1/whfatal"
	"gopkg.in/webhelp.v1/whlog"
	"gopkg.in/webhelp.v1/whmux"
	"gopkg.in/webhelp.v1/whredir"
	"gopkg.in/webhelp.v1/whroute"
	"gopkg.in/webhelp.v1/whsess"
)

var (
	listenAddr   = flag.String("addr", ":8080", "address to listen on")
	cookieSecret = flag.String("cookie_secret", "abcdef0123456789",
		"the secret for securing cookie information")

	projectId   = whmux.NewIntArg()
	controlId   = whmux.NewIntArg()
	sampleId    = whmux.NewIntArg()
	controlName = whmux.NewStringArg()
)

func main() {
	flag.Parse()
	loadOAuth2()
	secret, err := hex.DecodeString(*cookieSecret)
	if err != nil {
		panic(err)
	}

	renderer, err := NewRenderer()
	if err != nil {
		panic(err)
	}

	data, err := NewData()
	if err != nil {
		panic(err)
	}
	defer data.Close()

	endpoints := NewEndpoints(data)

	routes := whlog.LogRequests(whlog.Default, whfatal.Catch(
		whsess.HandlerWithStore(whsess.NewCookieStore(secret),
			whmux.Overlay{
				Default: endpoints.LoginRequired(whmux.Dir{
					"": whmux.Exact(renderer.Render(endpoints.ProjectList)),

					"project": projectId.ShiftOpt(
						whmux.Dir{
							"": whmux.Exact(renderer.Render(endpoints.Project)),

							"sample": sampleId.ShiftOpt(
								whmux.Dir{
									"": whmux.RequireGet(renderer.Render(endpoints.Sample)),
									"similar": whmux.RequireGet(
										renderer.Render(endpoints.SampleSimilar)),
								},
								whmux.ExactPath(whmux.Method{
									"GET": ProjectRedirector,
								}),
							),

							"control": controlId.ShiftOpt(
								whmux.Dir{
									"": whmux.Exact(renderer.Render(endpoints.Control)),
									"sample": whmux.ExactPath(whmux.RequireMethod("POST",
										renderer.Process(endpoints.NewSample))),
								},
								whmux.ExactPath(whmux.Method{
									"GET":  ProjectRedirector,
									"POST": renderer.Process(endpoints.NewControl),
								}),
							),

							"control_named": controlName.ShiftOpt(
								whmux.Dir{
									"sample": whmux.ExactPath(whmux.RequireMethod("POST",
										renderer.Process(endpoints.NewSampleFromName))),
								},
								whmux.RequireGet(ProjectRedirector),
							),

							"search": whmux.RequireMethod("POST",
								whmux.ExactPath(renderer.Render(endpoints.Search)),
							),
						},

						whmux.ExactPath(whmux.Method{
							"GET":  whredir.RedirectHandler("/"),
							"POST": renderer.Process(endpoints.NewProject),
						}),
					),

					"account": whmux.Dir{
						"apikeys": whmux.ExactPath(whmux.Method{
							"GET":  renderer.Render(endpoints.APIKeys),
							"POST": renderer.Process(endpoints.NewAPIKey),
						}),
					},
				}),
				Overlay: whmux.Dir{
					"auth": oauth2,
				}})))

	switch flag.Arg(0) {
	case "createdb":
		err := data.CreateDB()
		if err != nil {
			panic(err)
		}
	case "serve":
		panic(whlog.ListenAndServe(*listenAddr, routes))
	case "routes":
		whroute.PrintRoutes(os.Stdout, routes)
	default:
		fmt.Printf("Usage: %s <serve|createdb|routes>\n", os.Args[0])
	}
}
