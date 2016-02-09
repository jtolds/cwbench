// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jtolds/webhelp"
	"golang.org/x/net/context"
)

const (
	DefaultLimit = 25
)

type Endpoints struct {
	Data *Data
}

func NewEndpoints(data *Data) *Endpoints {
	return &Endpoints{Data: data}
}

func (a *Endpoints) APIKeys(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{}, err error) {
	keys, err := a.Data.APIKeys(user.Id)
	if err != nil {
		return "", nil, err
	}
	return "apikeys", map[string]interface{}{
		"Keys": keys,
	}, nil
}

func (a *Endpoints) NewAPIKey(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {

	_, err := a.Data.NewAPIKey(user.Id)
	if err != nil {
		return err
	}

	return webhelp.Redirect(w, req, fmt.Sprintf("/account/apikeys"))
}

func (a *Endpoints) ProjectList(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{},
	err error) {
	projects, err := a.Data.Projects(user.Id)
	if err != nil {
		return "", nil, err
	}
	return "projects", map[string]interface{}{
		"Projects": projects}, nil
}

func (a *Endpoints) Project(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{}, err error) {
	proj, read_only, err := a.Data.Project(user.Id, projectId.Get(ctx))
	if err != nil {
		return "", nil, webhelp.ErrNotFound.Wrap(err)
	}
	dimCount, diffexps, controls, err := a.Data.ProjectInfo(proj.Id)
	if err != nil {
		return "", nil, err
	}
	return "project", map[string]interface{}{
		"Project":        proj,
		"ReadOnly":       read_only,
		"DimensionCount": dimCount,
		"DiffExps":       diffexps,
		"Controls":       controls,
	}, nil
}

func (a *Endpoints) NewProject(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {
	proj_id, err := a.Data.NewProject(user.Id, req.FormValue("name"),
		func(deliver func(dim string) error) error {
			for _, dim := range strings.Fields(req.FormValue("dimensions")) {
				err := deliver(dim)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}
	return webhelp.Redirect(w, req, fmt.Sprintf("/project/%d", proj_id))
}

func (a *Endpoints) NewDiffExp(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {
	proj_id := projectId.Get(ctx)
	diffexp_id, err := a.Data.NewDiffExp(user.Id, proj_id, req.FormValue("name"),
		func(deliver func(dim_id int64, diff int) error) error {
			dimlookup, err := a.Data.DimLookup(proj_id)
			if err != nil {
				return err
			}

			for _, row := range strings.Split(req.FormValue("values"), "\n") {
				fields := strings.Fields(row)
				if len(fields) == 0 {
					continue
				}
				if len(fields) != 2 {
					return webhelp.ErrBadRequest.New("malformed data: %#v", row)
				}
				id, err := dimlookup.LookupId(fields[0])
				if err != nil {
					return err
				}
				val, err := strconv.Atoi(fields[1])
				if err != nil {
					return webhelp.ErrBadRequest.New("malformed data: %#v", row)
				}

				err = deliver(id, val)
				if err != nil {
					return err
				}
			}

			return nil
		})
	if err != nil {
		return err
	}
	return webhelp.Redirect(w, req, fmt.Sprintf("/project/%d/diffexp/%d",
		proj_id, diffexp_id))
}

func (a *Endpoints) DiffExp(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{}, err error) {
	proj, diffexp, err := a.Data.DiffExp(user.Id, projectId.Get(ctx),
		diffExpId.Get(ctx))
	if err != nil {
		return "", nil, webhelp.ErrNotFound.Wrap(err)
	}
	values, err := a.Data.DiffExpValues(diffexp.Id)
	if err != nil {
		return "", nil, err
	}
	dimlookup, err := a.Data.DimLookup(proj.Id)
	if err != nil {
		return "", nil, err
	}

	return "diffexp", map[string]interface{}{
		"Project": proj,
		"DiffExp": diffexp,
		"Values":  values,
		"Lookup":  dimlookup}, nil
}

func (a *Endpoints) DiffExpSimilar(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{}, err error) {
	proj, diffexp, err := a.Data.DiffExp(user.Id, projectId.Get(ctx),
		diffExpId.Get(ctx))
	if err != nil {
		return "", nil, webhelp.ErrNotFound.Wrap(err)
	}

	limit, err := strconv.Atoi(req.FormValue("k"))
	if err != nil {
		limit = DefaultLimit
	}

	up_regulated, down_regulated, err := a.Data.TopKSignature(diffexp.Id, limit)
	if err != nil {
		return "", nil, err
	}

	var results SearchResults
	search_type := req.FormValue("search-type")
	switch search_type {
	case "kolmogorov":
		results, err = a.Data.KSSearch(proj.Id, up_regulated, down_regulated)
	default:
		search_type = "topk"
		results, err = a.Data.TopKSearch(proj.Id, up_regulated, down_regulated,
			limit)
	}
	if err != nil {
		return "", nil, err
	}

	return "similar", map[string]interface{}{
		"Project": proj,
		"DiffExp": diffexp,
		"Results": results,
		"Params": url.Values{
			"k":           []string{fmt.Sprint(limit)},
			"search-type": []string{search_type}}.Encode()}, nil
}

func (a *Endpoints) Control(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{}, err error) {
	proj, control, read_only, err := a.Data.Control(user.Id, projectId.Get(ctx),
		controlId.Get(ctx))
	if err != nil {
		return "", nil, webhelp.ErrNotFound.Wrap(err)
	}
	values, err := a.Data.ControlValues(control.Id)
	if err != nil {
		return "", nil, err
	}
	dimlookup, err := a.Data.DimLookup(proj.Id)
	if err != nil {
		return "", nil, err
	}

	return "control", map[string]interface{}{
		"Project":  proj,
		"ReadOnly": read_only,
		"Control":  control,
		"Values":   values,
		"Lookup":   dimlookup}, nil
}

func (a *Endpoints) NewControl(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {
	proj_id := projectId.Get(ctx)
	control_id, err := a.Data.NewControl(user.Id, proj_id, req.FormValue("name"),
		func(deliver func(dim_id int64, value float64) error) error {
			dimlookup, err := a.Data.DimLookup(proj_id)
			if err != nil {
				return err
			}

			for _, row := range strings.Split(req.FormValue("values"), "\n") {
				fields := strings.Fields(row)
				if len(fields) == 0 {
					continue
				}
				if len(fields) != 2 {
					return webhelp.ErrBadRequest.New("malformed data: %#v", row)
				}
				id, err := dimlookup.LookupId(fields[0])
				if err != nil {
					return err
				}
				val, err := strconv.ParseFloat(fields[1], 64)
				if err != nil {
					return webhelp.ErrBadRequest.New("malformed data: %#v", row)
				}
				err = deliver(id, val)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}
	return webhelp.Redirect(w, req, fmt.Sprintf("/project/%d/control/%d",
		proj_id, control_id))
}

func (a *Endpoints) NewSample(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {
	return a.newSample(ctx, w, req, user, projectId.Get(ctx), controlId.Get(ctx))
}

func (a *Endpoints) NewSampleFromName(ctx context.Context,
	w webhelp.ResponseWriter, req *http.Request, user *UserInfo) error {
	proj_id := projectId.Get(ctx)
	control, err := a.Data.ControlByName(proj_id, controlName.Get(ctx))
	if err != nil {
		return err
	}
	return a.newSample(ctx, w, req, user, proj_id, control.Id)
}

func (a *Endpoints) newSample(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo, proj_id, control_id int64) error {

	diffexp_id, err := a.Data.NewSample(user.Id, proj_id, control_id,
		req.FormValue("name"),
		func(deliver func(dim_id int64, value float64) error) error {
			dimlookup, err := a.Data.DimLookup(proj_id)
			if err != nil {
				return err
			}
			for _, row := range strings.Split(req.FormValue("values"), "\n") {
				fields := strings.Fields(row)
				if len(fields) == 0 {
					continue
				}
				if len(fields) != 2 {
					return webhelp.ErrBadRequest.New("malformed data: %#v", row)
				}
				id, err := dimlookup.LookupId(fields[0])
				if err != nil {
					return err
				}
				val, err := strconv.ParseFloat(fields[1], 64)
				if err != nil {
					return webhelp.ErrBadRequest.New("malformed data: %#v", row)
				}
				err = deliver(id, val)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}
	return webhelp.Redirect(w, req, fmt.Sprintf("/project/%d/diffexp/%d",
		proj_id, diffexp_id))
}

func (a *Endpoints) Search(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{}, err error) {
	proj, _, err := a.Data.Project(user.Id, projectId.Get(ctx))
	if err != nil {
		return "", nil, webhelp.ErrNotFound.Wrap(err)
	}

	up_regulated_strings := strings.Fields(req.FormValue("up-regulated"))
	down_regulated_strings := strings.Fields(req.FormValue("down-regulated"))
	if len(up_regulated_strings)+len(down_regulated_strings) == 0 {
		return "", nil, webhelp.ErrBadRequest.New("no dimensions provided")
	}

	dimlookup, err := a.Data.DimLookup(proj.Id)
	if err != nil {
		return "", nil, err
	}

	seen := make(map[string]bool,
		len(up_regulated_strings)+len(down_regulated_strings))
	up_regulated := make([]int64, 0, len(up_regulated_strings))
	down_regulated := make([]int64, 0, len(down_regulated_strings))
	for _, val := range up_regulated_strings {
		if seen[val] {
			return "", nil, webhelp.ErrBadRequest.New("duplicated dimension")
		}
		seen[val] = true
		id, err := dimlookup.LookupId(val)
		if err != nil {
			return "", nil, err
		}
		up_regulated = append(up_regulated, id)
	}
	for _, val := range down_regulated_strings {
		if seen[val] {
			return "", nil, webhelp.ErrBadRequest.New("duplicated dimension")
		}
		seen[val] = true
		id, err := dimlookup.LookupId(val)
		if err != nil {
			return "", nil, err
		}
		down_regulated = append(down_regulated, id)
	}

	var results SearchResults
	switch req.FormValue("search-type") {
	case "kolmogorov":
		results, err = a.Data.KSSearch(proj.Id, up_regulated, down_regulated)
	case "topk":
		limit, err := strconv.Atoi(req.FormValue("k"))
		if err != nil {
			return "", nil, webhelp.ErrBadRequest.New("invalid k parameter")
		}
		results, err = a.Data.TopKSearch(proj.Id, up_regulated, down_regulated,
			limit)
	default:
		return "", nil, webhelp.ErrBadRequest.New("invalid search-type parameter")
	}
	if err != nil {
		return "", nil, err
	}

	return "results", map[string]interface{}{
		"Project": proj,
		"Results": results}, nil
}
