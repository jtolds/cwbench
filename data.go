// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"math"
	"sort"

	"github.com/jinzhu/gorm"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/errhttp"
)

var (
	sqlitePath = flag.String("db", "./db.db", "")

	Err         = errors.NewClass("error")
	ErrNotFound = Err.NewClass("not found", errhttp.SetStatusCode(404))
	ErrDenied   = Err.NewClass("denied", errhttp.SetStatusCode(405))
	ErrBadDims  = Err.NewClass("wrong dimensions", errhttp.SetStatusCode(400))
)

type Data struct {
	db gorm.DB
}

func NewData() (*Data, error) {
	db, err := gorm.Open("sqlite3", *sqlitePath)
	if err != nil {
		return nil, Err.Wrap(err)
	}
	return &Data{db: db}, nil
}

func (d *Data) DB() gorm.DB {
	return d.db
}

func (d *Data) Close() error {
	return Err.Wrap(d.db.Close())
}

func (d *Data) APIKeys(user_id string) (keys []*APIKey, err error) {
	return keys, Err.Wrap(d.db.Where("user_id = ?", user_id).Order(
		"key asc").Find(&keys).Error)
}

func (d *Data) APIKey(key string) (*APIKey, error) {
	var rv APIKey
	err := d.db.Where("key = ?", key).FirstOrInit(&rv).Error
	if err != nil {
		return nil, Err.Wrap(err)
	}
	if rv.UserId == "" {
		return nil, nil
	}
	return &rv, nil
}

func (d *Data) NewAPIKey(user_id string) (key string, err error) {
	var value [16]byte
	_, err = rand.Read(value[:])
	if err != nil {
		return "", err
	}
	key = hex.EncodeToString(value[:])

	tx := txWrapper{DB: d.db.Begin()}
	defer tx.Rollback()
	err = tx.Create(&APIKey{
		UserId: user_id,
		Key:    key}).Error
	if err != nil {
		return "", Err.Wrap(err)
	}

	tx.Commit()
	return key, nil
}

func (d *Data) Projects(user_id string) (rv []*Project, err error) {
	return rv, Err.Wrap(d.db.Where(
		"public OR user_id = ?", user_id).Order("name asc").Find(&rv).Error)
}

func (d *Data) Project(user_id string, project_id int64) (proj *Project,
	read_only bool, err error) {
	proj = &Project{}
	err = d.db.Where("(public OR user_id = ?) AND id = ?", user_id,
		project_id).First(proj).Error
	if err != nil {
		return nil, true, ErrNotFound.Wrap(err)
	}
	return proj, proj.UserId != user_id, nil
}

func (d *Data) ProjectInfo(project_id int64) (dimensions int,
	diffexps []DiffExp, controls []Control, err error) {
	dimensions, err = d.DimCount(project_id)
	if err != nil {
		return 0, nil, nil, err
	}
	err = d.db.Where("project_id = ?", project_id).Order("name asc").Find(
		&diffexps).Error
	if err != nil {
		return 0, nil, nil, Err.Wrap(err)
	}
	err = d.db.Where("project_id = ?", project_id).Order("name asc").Find(
		&controls).Error
	return dimensions, diffexps, controls, Err.Wrap(err)
}

func (d *Data) NewProject(user_id, name string,
	dimensions func(deliver func(dim string) error) error) (
	proj_id int64, err error) {
	tx := txWrapper{DB: d.db.Begin()}
	defer tx.Rollback()
	proj := Project{UserId: user_id, Name: name}
	err = tx.Create(&proj).Error
	if err != nil {
		return 0, Err.Wrap(err)
	}
	added := map[string]bool{}
	err = dimensions(func(dim string) error {
		if added[dim] {
			return ErrBadDims.New("duplicated dimension %#v", dim)
		}
		added[dim] = true
		return Err.Wrap(tx.Create(&Dimension{ProjectId: proj.Id, Name: dim}).Error)
	})
	if err != nil {
		return 0, err
	}
	tx.Commit()
	return proj.Id, nil
}

type DimLookup struct {
	db     gorm.DB
	projId int64

	dims     []Dimension
	nameToId map[string]int64
	idToName map[int64]string
}

func (d *DimLookup) loadDims() error {
	if d.dims != nil {
		return nil
	}
	err := d.db.Where("project_id = ?", d.projId).Find(&d.dims).Error
	if err != nil {
		d.dims = nil
		return Err.Wrap(err)
	}
	if d.dims == nil {
		d.dims = []Dimension{}
	}
	return nil
}

func (d *DimLookup) loadNameToId() error {
	if d.nameToId != nil {
		return nil
	}
	err := d.loadDims()
	if err != nil {
		return err
	}
	d.nameToId = make(map[string]int64, len(d.dims))
	for _, dim := range d.dims {
		d.nameToId[dim.Name] = dim.Id
	}
	return nil
}

func (d *DimLookup) loadIdToName() error {
	if d.idToName != nil {
		return nil
	}
	err := d.loadDims()
	if err != nil {
		return err
	}
	d.idToName = make(map[int64]string, len(d.dims))
	for _, dim := range d.dims {
		d.idToName[dim.Id] = dim.Name
	}
	return nil
}

func (d *DimLookup) LookupId(dim string) (id int64, err error) {
	err = d.loadNameToId()
	if err != nil {
		return 0, err
	}
	id, found := d.nameToId[dim]
	if !found {
		return 0, ErrBadDims.New("dimension missing")
	}
	return id, nil
}

func (d *DimLookup) LookupName(id int64) (name string, err error) {
	err = d.loadIdToName()
	if err != nil {
		return "", err
	}
	name, found := d.idToName[id]
	if !found {
		return "", ErrBadDims.New("dimension missing")
	}
	return name, nil
}

func (d *DimLookup) Count() (int, error) {
	err := d.loadDims()
	if err != nil {
		return 0, err
	}
	return len(d.dims), nil
}

func (d *Data) DimLookup(project_id int64) (*DimLookup, error) {
	return &DimLookup{db: d.db, projId: project_id}, nil
}

func (d *Data) DimCount(project_id int64) (rv int, err error) {
	return rv, Err.Wrap(d.db.Model(Dimension{}).Where("project_id = ?",
		project_id).Count(&rv).Error)
}

func (d *Data) AssertWriteAccess(user_id string, project_id int64,
	control_id *int64) (err error) {
	var read_only bool
	if control_id != nil {
		_, _, read_only, err = d.Control(user_id, project_id, *control_id)
	} else {
		_, read_only, err = d.Project(user_id, project_id)
	}
	if err != nil {
		return err
	}
	if read_only {
		return ErrDenied.New("read only project")
	}
	return nil
}

func (d *Data) DiffExp(user_id string, project_id, diff_exp_id int64) (
	*Project, *DiffExp, error) {
	var diffexp DiffExp
	err := d.db.Where("id = ?", diff_exp_id).First(&diffexp).Error
	if err != nil {
		return nil, nil, ErrNotFound.Wrap(err)
	}
	if project_id != diffexp.ProjectId {
		return nil, nil, ErrNotFound.New("not found")
	}
	proj, _, err := d.Project(user_id, diffexp.ProjectId)
	return proj, &diffexp, err
}

func (d *Data) DiffExpValues(diffexp_id int64) (
	values []DiffExpValue, err error) {
	return values, Err.Wrap(d.db.Where(
		"diff_exp_id = ?", diffexp_id).Order("diff desc").Find(&values).Error)
}

func (d *Data) NewDiffExp(user_id string, project_id int64, name string,
	values func(deliver func(dim_id int64, diff int) error) error) (
	diffexp_id int64, err error) {

	err = d.AssertWriteAccess(user_id, project_id, nil)
	if err != nil {
		return 0, err
	}

	tx := txWrapper{DB: d.db.Begin()}
	defer tx.Rollback()

	diffexp := DiffExp{ProjectId: project_id, Name: name}
	err = tx.Create(&diffexp).Error
	if err != nil {
		return 0, Err.Wrap(err)
	}

	count, err := d.DimCount(project_id)
	if err != nil {
		return 0, err
	}

	seen := make(map[int64]bool, count)
	err = values(func(dim_id int64, diff int) error {
		if seen[dim_id] {
			return ErrBadDims.New("duplicated dimension")
		}
		seen[dim_id] = true
		abs_diff := diff
		if abs_diff < 0 {
			abs_diff *= -1
		}
		return Err.Wrap(tx.Create(&DiffExpValue{
			DiffExpId:   diffexp.Id,
			DimensionId: dim_id,
			Diff:        diff,
			AbsDiff:     abs_diff}).Error)
	})
	if err != nil {
		return 0, err
	}

	if len(seen) != count {
		return 0, ErrBadDims.New("bad dimension count")
	}

	tx.Commit()
	return diffexp.Id, nil
}

func (d *Data) Control(user_id string, project_id, control_id int64) (
	proj *Project, control *Control, read_only bool, err error) {
	control = &Control{}
	err = d.db.Where("id = ?", control_id).First(control).Error
	if err != nil {
		return nil, nil, true, ErrNotFound.Wrap(err)
	}
	if project_id != control.ProjectId {
		return nil, nil, true, ErrNotFound.New("not found")
	}
	proj, read_only, err = d.Project(user_id, control.ProjectId)
	return proj, control, read_only, err
}

func (d *Data) ControlByName(project_id int64, name string) (
	control *Control, err error) {
	control = &Control{}
	err = ErrNotFound.Wrap(d.db.Where("project_id = ? AND name = ?",
		project_id, name).First(control).Error)
	if err != nil {
		return nil, err
	}
	return control, nil
}

func Ranked(count int,
	values func(deliver func(dim_id int64, value float64) error) error) (
	ranked func(deliver func(dim_id int64, rank int) error) error) {
	return func(deliver func(dim_id int64, rank int) error) error {
		seen := make(map[int64]bool, count)
		tosort := make(rankList, 0, count)

		err := values(func(dim_id int64, value float64) error {
			if seen[dim_id] {
				return ErrBadDims.New("duplicated dimension")
			}
			seen[dim_id] = true
			tosort = append(tosort, rankEntry{id: dim_id, val: value})
			return nil
		})
		if err != nil {
			return err
		}

		seen = nil
		if len(tosort) != count {
			return ErrBadDims.New(
				"submission dimensions don't match project dimensions")
		}

		return tosort.Rank(func(entry rankEntry, rank int) error {
			return deliver(entry.id, rank)
		})
	}
}

func (d *Data) NewControl(user_id string, project_id int64, name string,
	values func(deliver func(dim_id int64, value float64) error) error) (
	control_id int64, err error) {

	err = d.AssertWriteAccess(user_id, project_id, nil)
	if err != nil {
		return 0, err
	}

	tx := txWrapper{DB: d.db.Begin()}
	defer tx.Rollback()

	control := Control{ProjectId: project_id, Name: name}
	err = tx.Create(&control).Error
	if err != nil {
		return 0, Err.Wrap(err)
	}

	count, err := d.DimCount(project_id)
	if err != nil {
		return 0, err
	}

	err = Ranked(count, values)(func(dim_id int64, rank int) error {
		return Err.Wrap(tx.Create(&ControlValue{
			ControlId:   control.Id,
			DimensionId: dim_id,
			Rank:        rank}).Error)
	})
	if err != nil {
		return 0, err
	}

	tx.Commit()
	return control.Id, nil
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

func (d *Data) TopKSignature(diffexp_id int64, k int) (
	up, down []int64, err error) {
	var values []DiffExpValue
	err = d.db.Where("diff_exp_id = ?", diffexp_id).Order(
		"abs_diff desc").Limit(k).Find(&values).Error
	if err != nil {
		return nil, nil, Err.Wrap(err)
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

func (d *Data) ControlValues(control_id int64) (
	values []ControlValue, err error) {
	return values, Err.Wrap(d.db.Where(
		"control_id = ?", control_id).Order("rank desc").Find(&values).Error)
}

type SearchResult struct {
	DiffExp
	Score int
}

type SearchResults []SearchResult

func (l SearchResults) Len() int           { return len(l) }
func (l SearchResults) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l SearchResults) Less(i, j int) bool { return l[i].Score > l[j].Score }

func (d *Data) TopKSearch(proj_id int64, up, down []int64, k int) (
	result SearchResults, err error) {
	var diffexps []DiffExp
	err = d.db.Where("project_id = ?", proj_id).Find(&diffexps).Error
	if err != nil {
		return nil, Err.Wrap(err)
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
		other_up, other_down, err := d.TopKSignature(diffexp.Id, k)
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

func (d *Data) NewSample(user_id string, project_id, control_id int64,
	name string,
	values func(deliver func(dim_id int64, value float64) error) error) (
	diffexp_id int64, err error) {

	err = d.AssertWriteAccess(user_id, project_id, &control_id)
	if err != nil {
		return 0, err
	}

	control_values, err := d.ControlValues(control_id)
	if err != nil {
		return 0, err
	}

	control_lookup := make(map[int64]int, len(control_values))
	for _, val := range control_values {
		control_lookup[val.DimensionId] = val.Rank
	}
	control_values = nil

	return d.NewDiffExp(user_id, project_id, name,
		func(deliver func(dim_id int64, diff int) error) error {
			count, err := d.DimCount(project_id)
			if err != nil {
				return err
			}
			return Ranked(count, values)(func(dim_id int64, rank int) error {
				control_rank, exists := control_lookup[dim_id]
				if !exists {
					return ErrBadDims.New("dimension mismatch")
				}
				return deliver(dim_id, rank-control_rank)
			})
		})
}
