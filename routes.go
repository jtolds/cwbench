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
	diffExpId = webhelp.NewIntArgMux()
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

	app, err := NewApp()
	if err != nil {
		panic(err)
	}
	defer app.Close()

	routes := webhelp.LoggingHandler(
		sessions.HandlerWithStore(sessions.NewCookieStore(secret),
			webhelp.OverlayMux{
				Fallback: app.LoginRequired(webhelp.DirMux{
					"": webhelp.Exact(renderer.Render(app.ProjectList)),

					"project": projectId.OptShift(

						webhelp.ExactPath(webhelp.MethodMux{
							"GET":  webhelp.RedirectHandler("/"),
							"POST": renderer.Process(app.NewProject),
						}),

						webhelp.DirMux{
							"": webhelp.Exact(renderer.Render(app.Project)),

							"diffexp": diffExpId.OptShift(
								webhelp.ExactPath(webhelp.MethodMux{
									"GET":  ProjectRedirector,
									"POST": renderer.Process(app.NewDiffExp),
								}),
								webhelp.DirMux{
									"": webhelp.ExactGet(renderer.Render(app.DiffExp)),
									"similar": webhelp.ExactGet(
										renderer.Render(app.DiffExpSimilar)),
								},
							),

							"control": controlId.OptShift(
								webhelp.ExactPath(webhelp.MethodMux{
									"GET":  ProjectRedirector,
									"POST": renderer.Process(app.NewControl),
								}),

								webhelp.DirMux{
									"": webhelp.Exact(renderer.Render(app.Control)),
									"sample": webhelp.ExactPath(webhelp.ExactMethod("POST",
										renderer.Process(app.NewSample))),
								},
							),

							"search": webhelp.ExactMethod("POST",
								webhelp.ExactPath(renderer.Render(app.Search)),
							),
						},
					),

					"account": webhelp.DirMux{
						"apikeys": webhelp.ExactPath(webhelp.MethodMux{
							"GET":  renderer.Render(app.APIKeys),
							"POST": renderer.Process(app.NewAPIKey),
						}),
					},
				}),
				Overlay: webhelp.DirMux{
					"auth": oauth2,
				}}))

	switch flag.Arg(0) {
	case "createdb":
		err := app.CreateDB()
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
