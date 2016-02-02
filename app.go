// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"flag"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/jtolds/webhelp"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/context"
)

const (
	zeroWidth = math.SmallestNonzeroFloat64
)

var (
	sqlitePath = flag.String("db", "./db.db", "")
)

type App struct {
	db gorm.DB
}

func NewApp() (*App, error) {
	db, err := gorm.Open("sqlite3", *sqlitePath)
	if err != nil {
		return nil, err
	}
	return &App{db: db}, nil
}

func (a *App) Close() error { return a.db.Close() }

func (a *App) ProjectList(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{},
	err error) {
	var projects []*Project
	err = a.db.Where("public OR user_id = ?", user.Id).Order("name asc").Find(
		&projects).Error
	if err != nil {
		return "", nil, err
	}
	return "projects", map[string]interface{}{"Projects": projects}, nil
}

func (a *App) GetProject(user *UserInfo, project_id int64) (
	proj *Project, read_only bool, err error) {
	proj = &Project{}
	err = a.db.Where(
		"(public OR user_id = ?) AND id = ?", user.Id,
		project_id).First(proj).Error
	if err != nil {
		return nil, true, err
	}
	return proj, proj.UserId != user.Id, nil
}

func (a *App) GetDiffExp(user *UserInfo, project_id, diff_exp_id int64) (
	*Project, *DiffExp, error) {
	var diffexp DiffExp
	err := a.db.Where("id = ?", diff_exp_id).First(&diffexp).Error
	if err != nil {
		return nil, nil, err
	}
	if project_id != diffexp.ProjectId {
		return nil, nil, webhelp.ErrNotFound.New("not found")
	}
	proj, _, err := a.GetProject(user, diffexp.ProjectId)
	return proj, &diffexp, err
}

func (a *App) GetControl(user *UserInfo, project_id, control_id int64) (
	proj *Project, control *Control, read_only bool, err error) {
	control = &Control{}
	err = a.db.Where("id = ?", control_id).First(control).Error
	if err != nil {
		return nil, nil, true, err
	}
	if project_id != control.ProjectId {
		return nil, nil, true, webhelp.ErrNotFound.New("not found")
	}
	proj, read_only, err = a.GetProject(user, control.ProjectId)
	return proj, control, read_only, err
}

func (a *App) Project(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{}, err error) {
	proj, read_only, err := a.GetProject(user, projectId.Get(ctx))
	if err != nil {
		return "", nil, webhelp.ErrNotFound.Wrap(err)
	}
	var dimCount int
	err = a.db.Model(Dimension{}).Where("project_id = ?", proj.Id).Count(
		&dimCount).Error
	if err != nil {
		return "", nil, err
	}
	var diffexps []DiffExp
	err = a.db.Where("project_id = ?", proj.Id).Order("name asc").Find(
		&diffexps).Error
	if err != nil {
		return "", nil, err
	}
	var controls []Control
	err = a.db.Where("project_id = ?", proj.Id).Order("name asc").Find(
		&controls).Error
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

func (a *App) NewProject(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {
	tx := txWrapper{DB: a.db.Begin()}
	defer tx.Rollback()
	proj := Project{UserId: user.Id, Name: req.FormValue("name")}
	err := tx.Create(&proj).Error
	if err != nil {
		return err
	}
	added := map[string]bool{}
	for _, dim := range strings.Fields(req.FormValue("dimensions")) {
		if added[dim] {
			continue
		}
		added[dim] = true
		err := tx.Create(&Dimension{ProjectId: proj.Id, Name: dim}).Error
		if err != nil {
			return err
		}
	}
	tx.Commit()
	return webhelp.Redirect(w, req, fmt.Sprintf("/project/%d", proj.Id))
}

func (a *App) NewDiffExp(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {
	proj, read_only, err := a.GetProject(user, projectId.Get(ctx))
	if err != nil {
		return webhelp.ErrNotFound.Wrap(err)
	}
	if read_only {
		return webhelp.ErrMethodNotAllowed.New("read only project")
	}
	var dims []Dimension
	err = a.db.Where("project_id = ?", proj.Id).Find(&dims).Error
	if err != nil {
		return err
	}
	dimlookup := make(map[string]int64, len(dims))
	for _, dim := range dims {
		dimlookup[dim.Name] = dim.Id
	}
	dims = nil

	tx := txWrapper{DB: a.db.Begin()}
	defer tx.Rollback()

	diffexp := DiffExp{ProjectId: proj.Id, Name: req.FormValue("name")}
	err = tx.Create(&diffexp).Error
	if err != nil {
		return err
	}

	dimdiff := make(map[int64]int, len(dimlookup))

	for _, row := range strings.Split(req.FormValue("values"), "\n") {
		fields := strings.Fields(row)
		if len(fields) != 2 {
			return webhelp.ErrBadRequest.New("malformed data: %#v", row)
		}
		id, exists := dimlookup[fields[0]]
		if !exists {
			return webhelp.ErrBadRequest.New(
				"submission dimensions don't match project dimensions")
		}
		val, err := strconv.Atoi(fields[1])
		if err != nil {
			return webhelp.ErrBadRequest.New("malformed data: %#v", row)
		}

		if _, exists := dimdiff[id]; exists {
			return webhelp.ErrBadRequest.New("duplicated dimension")
		}
		dimdiff[id] = val
	}
	if len(dimdiff) != len(dimlookup) {
		return webhelp.ErrBadRequest.New(
			"submission dimensions don't match project dimensions")
	}
	dimlookup = nil

	for id, val := range dimdiff {
		absval := val
		if absval < 0 {
			absval *= -1
		}
		err := tx.Create(&DiffExpValue{
			DiffExpId:   diffexp.Id,
			DimensionId: id,
			Diff:        val,
			AbsDiff:     absval}).Error
		if err != nil {
			return err
		}
	}

	tx.Commit()
	return webhelp.Redirect(w, req, fmt.Sprintf("/project/%d/diffexp/%d",
		proj.Id, diffexp.Id))
}

func (a *App) DiffExp(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{}, err error) {
	proj, diffexp, err := a.GetDiffExp(user, projectId.Get(ctx),
		diffExpId.Get(ctx))
	if err != nil {
		return "", nil, webhelp.ErrNotFound.Wrap(err)
	}
	var values []DiffExpValue
	err = a.db.Where("diff_exp_id = ?", diffexp.Id).Order("diff desc").Find(
		&values).Error
	if err != nil {
		return "", nil, err
	}
	var dims []Dimension
	err = a.db.Where("project_id = ?", proj.Id).Find(&dims).Error
	if err != nil {
		return "", nil, err
	}
	dimlookup := make(map[int64]string, len(dims))
	for _, dim := range dims {
		dimlookup[dim.Id] = dim.Name
	}
	dims = nil

	return "diffexp", map[string]interface{}{
		"Project": proj,
		"DiffExp": diffexp,
		"Values":  values,
		"Lookup":  dimlookup}, nil
}

func (a *App) DiffExpSimilar(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{}, err error) {
	proj, diffexp, err := a.GetDiffExp(user, projectId.Get(ctx),
		diffExpId.Get(ctx))
	if err != nil {
		return "", nil, webhelp.ErrNotFound.Wrap(err)
	}

	limit, err := strconv.Atoi(req.FormValue("k"))
	if err != nil {
		return "", nil, webhelp.ErrBadRequest.New("missing numerical k query param")
	}

	up_regulated, down_regulated, err := a.topkSignature(ctx, diffexp, limit)
	if err != nil {
		return "", nil, err
	}

	results, err := a.topkSearch(ctx, proj, up_regulated, down_regulated, limit)
	if err != nil {
		return "", nil, err
	}

	return "similar", map[string]interface{}{
		"Project": proj,
		"DiffExp": diffexp,
		"Results": results}, nil
}

func (a *App) Control(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{}, err error) {
	proj, control, read_only, err := a.GetControl(user, projectId.Get(ctx),
		controlId.Get(ctx))
	if err != nil {
		return "", nil, webhelp.ErrNotFound.Wrap(err)
	}
	var values []ControlValue
	err = a.db.Where("control_id = ?", control.Id).Order(
		"rank desc").Find(&values).Error
	if err != nil {
		return "", nil, err
	}
	var dims []Dimension
	err = a.db.Where("project_id = ?", proj.Id).Find(&dims).Error
	if err != nil {
		return "", nil, err
	}
	dimlookup := make(map[int64]string, len(dims))
	for _, dim := range dims {
		dimlookup[dim.Id] = dim.Name
	}
	dims = nil

	return "control", map[string]interface{}{
		"Project":  proj,
		"ReadOnly": read_only,
		"Control":  control,
		"Values":   values,
		"Lookup":   dimlookup}, nil
}

func (a *App) NewControl(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {
	proj, read_only, err := a.GetProject(user, projectId.Get(ctx))
	if err != nil {
		return webhelp.ErrNotFound.Wrap(err)
	}
	if read_only {
		return webhelp.ErrMethodNotAllowed.New("read only project")
	}
	var dims []Dimension
	err = a.db.Where("project_id = ?", proj.Id).Find(&dims).Error
	if err != nil {
		return err
	}
	dimlookup := make(map[string]int64, len(dims))
	for _, dim := range dims {
		dimlookup[dim.Name] = dim.Id
	}
	dims = nil

	tx := txWrapper{DB: a.db.Begin()}
	defer tx.Rollback()

	control := Control{ProjectId: proj.Id, Name: req.FormValue("name")}
	err = tx.Create(&control).Error
	if err != nil {
		return err
	}

	seen := make(map[int64]bool, len(dimlookup))
	values := make(rankList, 0, len(dimlookup))

	for _, row := range strings.Split(req.FormValue("values"), "\n") {
		fields := strings.Fields(row)
		if len(fields) != 2 {
			return webhelp.ErrBadRequest.New("malformed data: %#v", row)
		}
		id, exists := dimlookup[fields[0]]
		if !exists {
			return webhelp.ErrBadRequest.New(
				"submission dimensions don't match project dimensions")
		}
		val, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return webhelp.ErrBadRequest.New("malformed data: %#v", row)
		}

		if seen[id] {
			return webhelp.ErrBadRequest.New("duplicated dimension")
		}
		seen[id] = true

		values = append(values, rankEntry{id: id, val: val})
	}
	seen = nil
	if len(values) != len(dimlookup) {
		return webhelp.ErrBadRequest.New(
			"submission dimensions don't match project dimensions")
	}
	dimlookup = nil

	err = values.Rank(func(entry rankEntry, rank int) error {
		return tx.Create(&ControlValue{
			ControlId:   control.Id,
			DimensionId: entry.id,
			Rank:        rank}).Error
	})
	if err != nil {
		return err
	}

	tx.Commit()
	return webhelp.Redirect(w, req, fmt.Sprintf("/project/%d/control/%d",
		proj.Id, control.Id))
}

type rankEntry struct {
	id  int64
	val float64
}
type rankList []rankEntry

func (l rankList) Len() int      { return len(l) }
func (l rankList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (p rankList) Less(i, j int) bool {
	return p[i].val < p[j].val || math.IsNaN(p[i].val) && !math.IsNaN(p[j].val)
}

func (p rankList) Rank(cb func(entry rankEntry, rank int) error) error {
	sort.Sort(p)
	last_rank := 1
	last_val := math.Inf(-1)
	for i, entry := range p {
		rank := i + 1
		if last_val >= entry.val {
			rank = last_rank
		} else {
			last_val = entry.val
			last_rank = rank
		}
		err := cb(entry, rank)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *App) NewSample(ctx context.Context, w webhelp.ResponseWriter,
	req *http.Request, user *UserInfo) error {
	proj, control, read_only, err := a.GetControl(user, projectId.Get(ctx),
		controlId.Get(ctx))
	if err != nil {
		return webhelp.ErrNotFound.Wrap(err)
	}
	if read_only {
		return webhelp.ErrMethodNotAllowed.New("read only project")
	}

	var control_values []ControlValue
	err = a.db.Where("control_id = ?", control.Id).Find(
		&control_values).Error
	if err != nil {
		return err
	}

	control_rank_lookup := make(map[int64]int, len(control_values))
	for _, val := range control_values {
		control_rank_lookup[val.DimensionId] = val.Rank
	}
	control_values = nil

	var dims []Dimension
	err = a.db.Where("project_id = ?", proj.Id).Find(&dims).Error
	if err != nil {
		return err
	}
	dimlookup := make(map[string]int64, len(dims))
	for _, dim := range dims {
		dimlookup[dim.Name] = dim.Id
		if _, exists := control_rank_lookup[dim.Id]; !exists {
			return webhelp.ErrInternalServerError.New("dimension id missing")
		}
	}
	dims = nil

	tx := txWrapper{DB: a.db.Begin()}
	defer tx.Rollback()

	diffexp := DiffExp{ProjectId: proj.Id, Name: req.FormValue("name")}
	err = tx.Create(&diffexp).Error
	if err != nil {
		return err
	}

	seen := make(map[int64]bool, len(dimlookup))
	values := make(rankList, 0, len(dimlookup))

	for _, row := range strings.Split(req.FormValue("values"), "\n") {
		fields := strings.Fields(row)
		if len(fields) != 2 {
			return webhelp.ErrBadRequest.New("malformed data: %#v", row)
		}
		id, exists := dimlookup[fields[0]]
		if !exists {
			return webhelp.ErrBadRequest.New(
				"submission dimensions don't match project dimensions")
		}
		val, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return webhelp.ErrBadRequest.New("malformed data: %#v", row)
		}

		if seen[id] {
			return webhelp.ErrBadRequest.New("duplicated dimension")
		}
		seen[id] = true

		values = append(values, rankEntry{id: id, val: val})
	}
	seen = nil
	if len(values) != len(dimlookup) {
		return webhelp.ErrBadRequest.New(
			"submission dimensions don't match project dimensions")
	}

	err = values.Rank(func(entry rankEntry, rank int) error {
		return tx.Create(&DiffExpValue{
			DiffExpId:   diffexp.Id,
			DimensionId: entry.id,
			Diff:        rank - control_rank_lookup[entry.id]}).Error
	})
	if err != nil {
		return err
	}

	tx.Commit()
	return webhelp.Redirect(w, req, fmt.Sprintf("/project/%d/diffexp/%d",
		proj.Id, diffexp.Id))
}

func (a *App) Search(ctx context.Context, req *http.Request,
	user *UserInfo) (tmpl string, page map[string]interface{}, err error) {
	proj, _, err := a.GetProject(user, projectId.Get(ctx))
	if err != nil {
		return "", nil, webhelp.ErrNotFound.Wrap(err)
	}

	up_regulated_strings := strings.Fields(req.FormValue("up-regulated"))
	down_regulated_strings := strings.Fields(req.FormValue("down-regulated"))
	limit, err := strconv.Atoi(req.FormValue("k"))
	if err != nil {
		return "", nil, webhelp.ErrBadRequest.New("invalid k parameter")
	}
	if len(up_regulated_strings)+len(down_regulated_strings) == 0 {
		return "", nil, webhelp.ErrBadRequest.New("no dimensions provided")
	}

	var dims []Dimension
	err = a.db.Where("project_id = ?", proj.Id).Find(&dims).Error
	if err != nil {
		return "", nil, err
	}
	dimlookup := make(map[string]int64, len(dims))
	for _, dim := range dims {
		dimlookup[dim.Name] = dim.Id
	}
	dims = nil

	seen := make(map[string]bool,
		len(up_regulated_strings)+len(down_regulated_strings))
	up_regulated := make([]int64, 0, len(up_regulated_strings))
	down_regulated := make([]int64, 0, len(down_regulated_strings))
	for _, val := range up_regulated_strings {
		if seen[val] {
			return "", nil, webhelp.ErrBadRequest.New("duplicated dimension")
		}
		seen[val] = true
		id, exists := dimlookup[val]
		if !exists {
			return "", nil, webhelp.ErrBadRequest.New("unknown dimension")
		}
		up_regulated = append(up_regulated, id)
	}
	for _, val := range down_regulated_strings {
		if seen[val] {
			return "", nil, webhelp.ErrBadRequest.New("duplicated dimension")
		}
		seen[val] = true
		id, exists := dimlookup[val]
		if !exists {
			return "", nil, webhelp.ErrBadRequest.New("unknown dimension")
		}
		down_regulated = append(down_regulated, id)
	}

	results, err := a.topkSearch(ctx, proj, up_regulated, down_regulated, limit)
	if err != nil {
		return "", nil, err
	}

	return "results", map[string]interface{}{
		"Project": proj,
		"Results": results}, nil
}

func (a *App) topkSignature(ctx context.Context, diffexp *DiffExp, limit int) (
	up, down []int64, err error) {

	var values []DiffExpValue
	err = a.db.Where("diff_exp_id = ?", diffexp.Id).Order(
		"abs_diff desc").Limit(limit).Find(&values).Error
	if err != nil {
		return nil, nil, err
	}

	for _, val := range values {
		if val.Diff < 0 {
			down = append(down, val.DimensionId)
		}
		if val.Diff > 0 {
			up = append(up, val.DimensionId)
		}
	}

	return up, down, nil
}

type SearchResult struct {
	DiffExp
	Score int
}

type SearchResults []SearchResult

func (l SearchResults) Len() int           { return len(l) }
func (l SearchResults) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l SearchResults) Less(i, j int) bool { return l[i].Score > l[j].Score }

func (a *App) topkSearch(ctx context.Context, proj *Project, up, down []int64,
	limit int) (result SearchResults, err error) {
	var diffexps []DiffExp
	err = a.db.Where("project_id = ?", proj.Id).Find(&diffexps).Error
	if err != nil {
		return nil, err
	}

	up_lookup := make(map[int64]bool, len(up))
	down_lookup := make(map[int64]bool, len(down))
	for _, id := range up {
		up_lookup[id] = true
	}
	for _, id := range down {
		down_lookup[id] = true
	}

	result = make(SearchResults, 0, len(diffexps))
	for _, diffexp := range diffexps {
		other_up, other_down, err := a.topkSignature(ctx, &diffexp, limit)
		if err != nil {
			return nil, err
		}
		val := SearchResult{DiffExp: diffexp}
		for _, id := range other_up {
			if up_lookup[id] {
				val.Score += 1
			}
		}
		for _, id := range other_down {
			if down_lookup[id] {
				val.Score += 1
			}
		}
		result = append(result, val)
	}

	sort.Sort(result)

	return result, nil
}
