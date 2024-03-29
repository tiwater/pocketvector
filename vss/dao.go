package vss

import (
	"context"
	"database/sql"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
)

type VssDao struct {
	daos.Dao
}

func NewDao(app core.App) (*VssDao, error) {

	maxOpenConns := core.DefaultDataMaxOpenConns
	maxIdleConns := core.DefaultDataMaxIdleConns

	concurrentDB, err := connectDB(filepath.Join(app.DataDir(), "vss.db"))
	if err != nil {
		return nil, err
	}
	concurrentDB.DB().SetMaxOpenConns(maxOpenConns)
	concurrentDB.DB().SetMaxIdleConns(maxIdleConns)
	concurrentDB.DB().SetConnMaxIdleTime(5 * time.Minute)

	nonconcurrentDB, err := connectDB(filepath.Join(app.DataDir(), "vss.db"))
	if err != nil {
		return nil, err
	}
	nonconcurrentDB.DB().SetMaxOpenConns(1)
	nonconcurrentDB.DB().SetMaxIdleConns(1)
	nonconcurrentDB.DB().SetConnMaxIdleTime(5 * time.Minute)

	if app.IsDev() {
		nonconcurrentDB.QueryLogFunc = func(ctx context.Context, t time.Duration, sql string, rows *sql.Rows, err error) {
			color.HiBlack("[%.2fms] %v\n", float64(t.Milliseconds()), sql)
		}
		concurrentDB.QueryLogFunc = nonconcurrentDB.QueryLogFunc

		nonconcurrentDB.ExecLogFunc = func(ctx context.Context, t time.Duration, sql string, result sql.Result, err error) {
			color.HiBlack("[%.2fms] %v\n", float64(t.Milliseconds()), sql)
		}
		concurrentDB.ExecLogFunc = nonconcurrentDB.ExecLogFunc
	}

	dao := daos.NewMultiDB(concurrentDB, nonconcurrentDB)

	vssDao := VssDao{}
	vssDao.Dao = *dao

	return &vssDao, nil
}

// ResetDB takes care for releasing initialized app resources
// (eg. closing db connections).
func (dao *VssDao) ResetDB() error {
	if err := dao.ConcurrentDB().(*dbx.DB).Close(); err != nil {
		return err
	}
	if err := dao.NonconcurrentDB().(*dbx.DB).Close(); err != nil {
		return err
	}

	return nil
}
