package vss

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

func isVssRecord(record *models.Record) bool {
	dim := record.Collection().Options.Get(KeyDimension)
	return dim != nil
}

func RegisterHooks(app core.App, vssDao *VssDao) {
	// After the record is creatd, so that we can get the rowid
	app.OnRecordAfterCreateRequest().Add(func(e *core.RecordCreateEvent) error {
		q := VssQuery{
			Dao:    app.Dao(),
			VssDao: vssDao,
		}
		if !isVssRecord(e.Record) {
			return nil
		}
		if err := q.Insert(e.Record); err != nil {
			app.Logger().Error(err.Error())
			return err
		}
		return nil
	})

	app.OnRecordAfterUpdateRequest().Add(func(e *core.RecordUpdateEvent) error {
		q := VssQuery{
			Dao:    app.Dao(),
			VssDao: vssDao,
		}
		if !isVssRecord(e.Record) {
			return nil
		}
		if err := q.Update(e.Record); err != nil {
			app.Logger().Error(err.Error())
			return err
		}
		return nil
	})

	// Before the record is deleted, so that we can get the rowid
	app.OnRecordBeforeDeleteRequest().Add(func(e *core.RecordDeleteEvent) error {
		q := VssQuery{
			Dao:    app.Dao(),
			VssDao: vssDao,
		}
		if !isVssRecord(e.Record) {
			return nil
		}
		if err := q.Delete(e.Record); err != nil {
			app.Logger().Error(err.Error())
			return err
		}
		return nil
	})
}
