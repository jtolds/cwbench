// Copyright (C) 2016 JT Olds
// See LICENSE for copying information

package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"math"
	"runtime"
	"sort"
	"sync"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/errhttp"
)

var (
	dbType = flag.String("db", "postgres", "the database type to use. "+
		"can be postgres or sqlite3")
	dbConn = flag.String("db.conn",
		"user=cwbench database=cwbench password=password",
		"the database connection string")
	searchParallelism = flag.Int("parallelism", runtime.NumCPU()+1,
		"number of parellel search queries to execute")

	Err         = errors.NewClass("error")
	ErrNotFound = Err.NewClass("not found", errhttp.SetStatusCode(404))
	ErrDenied   = Err.NewClass("denied", errhttp.SetStatusCode(405))
	ErrBadDims  = Err.NewClass("wrong dimensions", errhttp.SetStatusCode(400))
)

type Data struct {
	db gorm.DB
}

func NewData() (*Data, error) {
	db, err := gorm.Open(*dbType, *dbConn)
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
	samples []Sample, controls []Control, err error) {
	dimensions, err = d.DimCount(project_id)
	if err != nil {
		return 0, nil, nil, err
	}
	err = d.db.Where("project_id = ?", project_id).Order("name asc").Find(
		&samples).Error
	if err != nil {
		return 0, nil, nil, Err.Wrap(err)
	}
	err = d.db.Where("project_id = ?", project_id).Order("name asc").Find(
		&controls).Error
	return dimensions, samples, controls, Err.Wrap(err)
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

func (d *Data) Sample(user_id string, project_id, sample_id int64) (
	*Project, *Sample, error) {
	var sample Sample
	err := d.db.Where("id = ?", sample_id).First(&sample).Error
	if err != nil {
		return nil, nil, ErrNotFound.Wrap(err)
	}
	if project_id != sample.ProjectId {
		return nil, nil, ErrNotFound.New("not found")
	}
	proj, _, err := d.Project(user_id, sample.ProjectId)
	return proj, &sample, err
}

func (d *Data) SampleValues(sample_id int64) (
	values []SampleValue, err error) {
	return values, Err.Wrap(d.db.Where(
		"sample_id = ?", sample_id).Order("rank_diff desc").Find(&values).Error)
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
	values func(func(dim_id int64, value float64) error) error) (
	ranked func(func(dim_id int64, value float64, rank int) error) error) {
	return func(deliv func(dim_id int64, value float64, rank int) error) error {
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

		return tosort.Rank(func(entry rankEntry, value float64, rank int) error {
			return deliv(entry.id, value, rank)
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

	err = Ranked(count, values)(
		func(dim_id int64, value float64, rank int) error {
			return Err.Wrap(tx.Create(&ControlValue{
				ControlId:   control.Id,
				DimensionId: dim_id,
				Value:       value,
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

func (p rankList) Rank(
	cb func(entry rankEntry, value float64, rank int) error) error {
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
		err := cb(entry, entry.val, rank)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Data) ControlValues(control_id int64) (
	values []ControlValue, err error) {
	return values, Err.Wrap(d.db.Where(
		"control_id = ?", control_id).Order("rank desc").Find(&values).Error)
}

func (d *Data) NewSample(user_id string, project_id, control_id int64,
	name string,
	values func(deliver func(dim_id int64, value float64) error) error) (
	sample_id int64, err error) {

	err = d.AssertWriteAccess(user_id, project_id, &control_id)
	if err != nil {
		return 0, err
	}

	control_values, err := d.ControlValues(control_id)
	if err != nil {
		return 0, err
	}

	control_lookup := make(map[int64]*ControlValue, len(control_values))
	for i := range control_values {
		control_lookup[control_values[i].DimensionId] = &control_values[i]
	}

	tx := txWrapper{DB: d.db.Begin()}
	defer tx.Rollback()

	sample := Sample{ProjectId: project_id, Name: name, ControlId: control_id}
	err = tx.Create(&sample).Error
	if err != nil {
		return 0, Err.Wrap(err)
	}

	seen := make(map[int64]bool, len(control_values))

	err = Ranked(len(control_values), values)(
		func(dim_id int64, value float64, rank int) error {
			if seen[dim_id] {
				return ErrBadDims.New("duplicated dimension")
			}
			seen[dim_id] = true

			control, exists := control_lookup[dim_id]
			if !exists {
				return ErrBadDims.New("dimension mismatch")
			}

			rank_diff := rank - control.Rank
			abs_rank_diff := rank_diff
			if abs_rank_diff < 0 {
				abs_rank_diff *= -1
			}

			value_diff := value - control.Value
			abs_value_diff := value_diff
			if abs_value_diff < 0 {
				abs_value_diff *= -1
			}
			return Err.Wrap(tx.Create(&SampleValue{
				SampleId:    sample.Id,
				DimensionId: dim_id,

				Rank:        rank,
				RankDiff:    rank_diff,
				AbsRankDiff: abs_rank_diff,

				Value:        value,
				ValueDiff:    value_diff,
				AbsValueDiff: abs_value_diff,
			}).Error)
		})
	if err != nil {
		return 0, err
	}

	if len(seen) != len(control_values) {
		return 0, ErrBadDims.New("bad dimension count")
	}

	tx.Commit()
	return sample.Id, nil
}

type TopKType string

const (
	TopKRankDiff  TopKType = "abs_rank_diff desc"
	TopKValueDiff TopKType = "abs_value_diff desc"
)

func (d *Data) TopKSignature(sample_id int64, k int,
	top_k_type TopKType) (up, down []int64, err error) {
	var values []SampleValue

	err = d.db.Where("sample_id = ?", sample_id).Order(
		string(top_k_type)).Limit(k).Find(&values).Error
	if err != nil {
		return nil, nil, Err.Wrap(err)
	}

	switch top_k_type {
	case TopKRankDiff:
		for _, val := range values {
			if val.RankDiff < 0 {
				down = append(down, val.DimensionId)
			} else if val.RankDiff > 0 {
				up = append(up, val.DimensionId)
			}
		}
	case TopKValueDiff:
		for _, val := range values {
			if val.ValueDiff < 0 {
				down = append(down, val.DimensionId)
			} else if val.ValueDiff > 0 {
				up = append(up, val.DimensionId)
			}
		}
	}

	return up, down, nil
}

type SearchResult struct {
	Sample
	Score float64
}

type SearchResults []SearchResult

func (l SearchResults) Len() int           { return len(l) }
func (l SearchResults) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l SearchResults) Less(i, j int) bool { return l[i].Score > l[j].Score }

func (d *Data) search(proj_id int64,
	scoreFunc func(sample_id int64) (score float64, err error)) (
	SearchResults, error) {

	var samples []Sample
	err := d.db.Where("project_id = ?", proj_id).Find(&samples).Error
	if err != nil {
		return nil, Err.Wrap(err)
	}

	var wg sync.WaitGroup
	var result_mtx sync.Mutex
	result := make(SearchResults, 0, len(samples))
	var result_errs errors.ErrorGroup
	samples_ch := make(chan Sample)

	wg.Add(*searchParallelism)
	for i := 0; i < *searchParallelism; i++ {
		go func() {
			defer wg.Done()
			for sample := range samples_ch {
				val, err := scoreFunc(sample.Id)
				result_mtx.Lock()
				result_errs.Add(err)
				if err == nil {
					result = append(result, SearchResult{Sample: sample, Score: val})
				}
				result_mtx.Unlock()
			}
		}()
	}

	for _, sample := range samples {
		samples_ch <- sample
	}
	close(samples_ch)
	wg.Wait()

	err = result_errs.Finalize()
	if err != nil {
		return nil, err
	}

	sort.Sort(result)
	return result, nil
}

func (d *Data) TopKSearch(proj_id int64, up, down []int64, k int,
	top_k_type TopKType) (result SearchResults, err error) {
	up_lookup := make(map[int64]bool, len(up))
	down_lookup := make(map[int64]bool, len(down))
	for _, id := range up {
		up_lookup[id] = true
	}
	for _, id := range down {
		down_lookup[id] = true
	}
	return d.search(proj_id, func(sample_id int64) (float64, error) {
		other_up, other_down, err := d.TopKSignature(sample_id, k, top_k_type)
		if err != nil {
			return math.NaN(), err
		}
		val := 0
		for _, id := range other_up {
			if up_lookup[id] {
				val += 1
			}
			if down_lookup[id] {
				val -= 1
			}
		}
		for _, id := range other_down {
			if up_lookup[id] {
				val -= 1
			}
			if down_lookup[id] {
				val += 1
			}
		}
		return float64(val), nil
	})
}

func (d *Data) KSSearch(proj_id int64, up, down []int64) (
	result SearchResults, err error) {
	return nil, errors.NotImplementedError.New("TODO")

	up_lookup := make(map[int64]bool, len(up))
	down_lookup := make(map[int64]bool, len(down))
	for _, id := range up {
		up_lookup[id] = true
	}
	for _, id := range down {
		down_lookup[id] = true
	}

	return d.search(proj_id, func(sample_id int64) (float64, error) {
		var values []SampleValue
		err = d.db.Where("sample_id = ?", sample_id).Order(
			"diff desc").Find(&values).Error
		if err != nil {
			return math.NaN(), err
		}
		distance := 0
		max_distance := math.Inf(-1)
		for _, val := range values {
			if val.RankDiff >= 0 {
				if up_lookup[val.DimensionId] {
					distance += 1
				} else {
					distance -= 1
				}
			} else {
				if down_lookup[val.DimensionId] {
					distance += 1
				} else {
					distance -= 1
				}
			}
			newdist := math.Abs(float64(distance))
			if newdist > max_distance {
				max_distance = newdist
			}
		}
		return max_distance, nil
	})
}
