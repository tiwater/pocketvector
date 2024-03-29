// sqlite-vss related operations
package vss

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/resolvers"
	"github.com/pocketbase/pocketbase/tools/search"
)

const (
	TableNamePrefix  = "vss_"
	KeyDimension     = "vss_dimension"
	ExtensionVector0 = "vector0"
	ExtensionVss0    = "vss0"
)

type VssQuery struct {
	Dao    *daos.Dao
	VssDao *VssDao
}

func (q VssQuery) Insert(record *models.Record) error {
	vssTableName := vssTableName(record.Collection())
	if !vssTableExists(q.VssDao.DB(), vssTableName) {
		if err := createVssTable(q.VssDao.DB(), record.Collection(), []string{}); err != nil {
			return err
		}
	}
	rowid, err := modelRowId(q.Dao.DB(), record)
	if err != nil {
		return err
	}
	cols := columns(record, []string{})
	// Set rowid
	cols["rowid"] = rowid
	_, err = q.VssDao.DB().Insert(vssTableName, dbx.Params(cols)).Execute()

	return err
}

func (q VssQuery) Update(record *models.Record) error {
	vssTableName := vssTableName(record.Collection())
	if !vssTableExists(q.VssDao.DB(), vssTableName) {
		return q.Insert(record)
	}
	rowid, err := modelRowId(q.Dao.DB(), record)
	if err != nil {
		return err
	}
	cols := columns(record, []string{})
	_, err = q.VssDao.DB().Update(vssTableName, dbx.Params(cols), dbx.HashExp{"rowid": rowid}).Execute()

	return nil
}

func (q VssQuery) Delete(record *models.Record) error {
	vssTableName := vssTableName(record.Collection())
	if !vssTableExists(q.VssDao.DB(), vssTableName) {
		return nil
	}
	rowid, err := modelRowId(q.Dao.DB(), record)
	if err != nil {
		return err
	}
	_, err = q.VssDao.DB().Delete(vssTableName, dbx.HashExp{"rowid": rowid}).Execute()
	return err
}

type VssResult struct {
	TotalItems int             `json:"totalItems"`
	Items      []VssResultItem `json:"items"`
}

type VssResultItem struct {
	Distance float64        `json:"distance"`
	Record   *models.Record `json:"item"`
}

func (q VssQuery) Search(requestInfo *models.RequestInfo, collection *models.Collection, field string, criteria string, limit int64) (*VssResult, error) {
	vssTableName := vssTableName(collection)

	vss := fmt.Sprintf("vss_search(%s, '%s')", field, criteria)
	type RowidDistance struct {
		Rowid    int64   `db:"rowid"`
		Distance float64 `db:"distance"`
	}
	var rowidDistances []RowidDistance
	if err := q.VssDao.DB().Select("rowid", "distance").From(vssTableName).Where(dbx.NewExp(vss)).Limit(limit).All(&rowidDistances); err != nil {
		return nil, err
	}

	var mapRowid2Distance = make(map[int64]float64)
	for _, v := range rowidDistances {
		mapRowid2Distance[v.Rowid] = v.Distance
	}

	rowids := make([]interface{}, len(rowidDistances))
	for i, v := range rowidDistances {
		rowids[i] = v.Rowid
	}

	// Expand the real record

	type RowidId struct {
		Id    string `db:"id"`
		Rowid int64  `db:"rowid"`
	}
	var rowidIds []RowidId
	if err := q.Dao.DB().Select("id", "rowid").From(collection.Name).Where(dbx.In("rowid", rowids...)).All(&rowidIds); err != nil {
		return nil, err
	}
	var mapId2Rowid = make(map[string]int64)
	for _, v := range rowidIds {
		mapId2Rowid[v.Id] = v.Rowid
	}

	fieldsResolver := resolvers.NewRecordFieldResolver(
		q.Dao,
		collection,
		requestInfo,
		// hidden fields are searchable only by admins
		requestInfo.Admin != nil,
	)

	searchProvider := search.NewProvider(fieldsResolver).
		Query(q.Dao.RecordQuery(collection))

	if requestInfo.Admin == nil && collection.ListRule != nil {
		searchProvider.AddFilter(search.FilterData(*collection.ListRule))
	}
	if len(rowidIds) > 0 {
		idFilters := make([]string, len(rowidIds))
		for i, v := range rowidIds {
			idFilters[i] = fmt.Sprintf("id='%s'", v.Id)
		}
		searchProvider.AddFilter(search.FilterData(strings.Join(idFilters, "||")))
	}

	records := []*models.Record{}

	_, err := searchProvider.Exec(&records)
	if err != nil {
		return nil, err
	}

	resultItems := make([]VssResultItem, len(records))

	for i, v := range records {
		resultItems[i].Record = v
		resultItems[i].Distance = mapRowid2Distance[mapId2Rowid[v.Id]]
	}

	result := VssResult{
		TotalItems: len(rowidDistances),
		Items:      resultItems,
	}

	return &result, nil
}

func vssTableName(collection *models.Collection) string {
	return TableNamePrefix + collection.Name
}

func vssTableExists(db dbx.Builder, tableName string) bool {
	var v []string
	err := db.Select("name").From("sqlite_master").Where(dbx.HashExp{"type": "table", "name": tableName}).Column(&v)
	return err == nil && len(v) > 0
}

func columns(record *models.Record, exclude []string) map[string]interface{} {
	cols := make(map[string]interface{})
	for _, i := range record.Collection().Schema.Fields() {
		if i.Type == schema.FieldTypeJson && !slices.Contains(exclude, i.Name) {
			if v := record.Get(i.Name); v != nil {
				cols[i.Name] = v
			}
		}
	}
	return cols
}

func modelRowId(db dbx.Builder, record *models.Record) (int64, error) {
	var rowids []int64
	err := db.Select("rowid").From(record.TableName()).Where(dbx.HashExp{"id": record.Id}).Column(&rowids)
	return rowids[0], err
}

func vssFields(collection *models.Collection, exclude []string) []string {
	fields := []string{}

	for _, i := range collection.Schema.Fields() {
		if i.Type == schema.FieldTypeJson && !slices.Contains(exclude, i.Name) {
			fields = append(fields, i.Name)
		}
	}
	return fields
}

func createVssTable(db dbx.Builder, collection *models.Collection, exclude []string) error {
	tableName := vssTableName(collection)
	fields := vssFields(collection, exclude)
	dim := collection.Options.Get(KeyDimension).(float64)
	fds := make([]string, len(fields))
	for i, v := range fields {
		fds[i] = fmt.Sprintf("%s(%d)", v, int(dim))
	}
	sql := fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS %s using vss0(%s)", tableName, strings.Join(fds, ","))
	_, err := db.NewQuery(sql).Execute()
	return err
}
