package apis

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/core"
	"github.com/tiwater/pocketvector/vss"
)

func BindApis(app core.App, dao *vss.VssDao, rg *echo.Group) {
	bindVssApi(app, dao, rg)
}
