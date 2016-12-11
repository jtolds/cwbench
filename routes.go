// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/jtolds/webhelp"
	"github.com/jtolds/webhelp/sessions"
)

var (
	listenAddr   = flag.String("addr", ":8080", "address to listen on")
	cookieSecret = flag.String("cookie_secret", "abcdef0123456789",
		"the secret for securing cookie information")

	projectId = webhelp.NewIntArgMux()
	controlId = webhelp.NewIntArgMux()
	sampleId  = webhelp.NewIntArgMux()

	controlName = webhelp.NewStringArgMux()
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

	routes := webhelp.LoggingHandler(webhelp.FatalHandler(
		sessions.HandlerWithStore(sessions.NewCookieStore(secret),
			webhelp.OverlayMux{
				Default: endpoints.LoginRequired(webhelp.DirMux{
					"": webhelp.Exact(renderer.Render(endpoints.ProjectList)),

					"project": projectId.ShiftOpt(
						webhelp.DirMux{
							"": webhelp.Exact(renderer.Render(endpoints.Project)),

							"sample": sampleId.ShiftOpt(
								webhelp.DirMux{
									"": webhelp.RequireGet(renderer.Render(endpoints.Sample)),
									"similar": webhelp.RequireGet(
										renderer.Render(endpoints.SampleSimilar)),
								},
								webhelp.ExactPath(webhelp.MethodMux{
									"GET": ProjectRedirector,
								}),
							),

							"control": controlId.ShiftOpt(
								webhelp.DirMux{
									"": webhelp.Exact(renderer.Render(endpoints.Control)),
									"sample": webhelp.ExactPath(webhelp.RequireMethod("POST",
										renderer.Process(endpoints.NewSample))),
								},
								webhelp.ExactPath(webhelp.MethodMux{
									"GET":  ProjectRedirector,
									"POST": renderer.Process(endpoints.NewControl),
								}),
							),

							"control_named": controlName.ShiftOpt(
								webhelp.DirMux{
									"sample": webhelp.ExactPath(webhelp.RequireMethod("POST",
										renderer.Process(endpoints.NewSampleFromName))),
								},
								webhelp.RequireGet(ProjectRedirector),
							),

							"search": webhelp.RequireMethod("POST",
								webhelp.ExactPath(renderer.Render(endpoints.Search)),
							),
						},

						webhelp.ExactPath(webhelp.MethodMux{
							"GET":  webhelp.RedirectHandler("/"),
							"POST": renderer.Process(endpoints.NewProject),
						}),
					),

					"account": webhelp.DirMux{
						"apikeys": webhelp.ExactPath(webhelp.MethodMux{
							"GET":  renderer.Render(endpoints.APIKeys),
							"POST": renderer.Process(endpoints.NewAPIKey),
						}),
					},
				}),
				Overlay: webhelp.DirMux{
					"auth": oauth2,
				}})))

	switch flag.Arg(0) {
	case "createdb":
		err := data.CreateDB()
		if err != nil {
			panic(err)
		}
	case "serve":
		panic(webhelp.ListenAndServe(*listenAddr, routes))
	case "routes":
		webhelp.PrintRoutes(os.Stdout, routes)
	default:
		fmt.Printf("Usage: %s <serve|createdb|routes>\n", os.Args[0])
	}
}
