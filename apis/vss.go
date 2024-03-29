package apis

import (
	"encoding/json"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/tiwater/pocketvector/vss"
	"golang.org/x/exp/maps"
)

const (
	MaxSearchResult = 100
)

func bindVssApi(app core.App, vssDao *vss.VssDao, rg *echo.Group) {
	api := vssApi{app: app, vssDao: vssDao}
	subGroup := rg.Group(
		"/collections/:collection",
		apis.ActivityLogger(app),
	)

	subGroup.POST("/vss", api.vssSearch, apis.LoadCollectionContext(app, models.CollectionTypeBase, models.CollectionTypeAuth))
}

type vssApi struct {
	app    core.App
	vssDao *vss.VssDao
}

type searchRequests = map[string]interface{}

func (api *vssApi) vssSearch(c echo.Context) error {
	collection, _ := c.Get(apis.ContextCollectionKey).(*models.Collection)
	if collection == nil {
		return apis.NewNotFoundError("", "Missing collection context.")
	}

	requestInfo := apis.RequestInfo(c)
	var limit int64 = MaxSearchResult
	if v, ok := requestInfo.Query["limit"]; ok {
		limit = min(v.(int64), MaxSearchResult)
	}

	searchParams := make(searchRequests)

	if err := c.Bind(&searchParams); err != nil {
		return err
	}

	if len(searchParams) > 2 {
		api.app.Logger().Debug(strings.Join(maps.Keys(searchParams), ","))
		return apis.NewBadRequestError("Can only search by one parameter", nil)
	}

	q := vss.VssQuery{
		Dao:    api.app.Dao(),
		VssDao: api.vssDao,
	}
	for k, v := range searchParams {
		if k == "collection" {
			continue
		}
		js, _ := json.Marshal(v)
		results, err := q.Search(requestInfo, collection, k, string(js), limit)
		if err != nil {
			return err
		}
		return c.JSON(200, results)
	}

	return nil
}
