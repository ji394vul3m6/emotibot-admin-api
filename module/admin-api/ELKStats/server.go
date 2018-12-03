package ELKStats

import (
	"fmt"
	"net/http"
	"time"

	"emotibot.com/emotigo/module/admin-api/ELKStats/controllers"
	"emotibot.com/emotigo/module/admin-api/ELKStats/services"
	"emotibot.com/emotigo/module/admin-api/util"
	"emotibot.com/emotigo/pkg/api/dal/v1"
)

var (
	// ModuleInfo is needed for module define
	ModuleInfo util.ModuleInfo
)

// Init init the package ModuleInfo & other essential data
func Init() error {
	var moduleName = "stats"
	var dalClient *dal.Client
	var err error

	dalAddress, ok := util.GetEnvOf("server")["DAL_URL"]
	if ok {
		var httpClient = &http.Client{
			Timeout: time.Duration(5) * time.Second,
		}
		dalClient, err = dal.NewClientWithHTTPClient(dalAddress, httpClient)
		if err != nil {
			err = fmt.Errorf("init dal client failed, %v", err)
		}
	} else {
		err = fmt.Errorf("Require Module Env DAL_URL")
	}
	if err != nil {
		return err
	}

	ModuleInfo = util.ModuleInfo{
		ModuleName: moduleName,
		EntryPoints: []util.EntryPoint{
			util.NewEntryPoint("GET", "visit", []string{"view"}, controllers.VisitStatsGetHandler),
			util.NewEntryPoint("GET", "question", []string{"view"}, controllers.QuestionStatsGetHandler),
			util.NewEntryPoint("POST", "records/query", []string{"view"}, controllers.VisitRecordsGetHandler),
			util.NewEntryPoint("POST", "records/export", []string{"view", "export"}, controllers.VisitRecordsExportHandler),
			util.NewEntryPoint("GET", "records/export/{export_id}",
				[]string{"view", "export"}, controllers.VisitRecordsExportDownloadHandler),
			util.NewEntryPoint("DELETE", "records/export/{export_id}",
				[]string{"view", "export"}, controllers.VisitRecordsExportDeleteHandler),
			util.NewEntryPoint("GET", "records/export/{export_id}/status",
				[]string{"view", "export"}, controllers.VisitRecordsExportStatusHandler),
			util.NewEntryPoint("POST", "records/mark", []string{"view", "export"}, controllers.NewRecordsMarkUpdateHandler(dalClient)),
			util.NewEntryPoint("POST", "records/ignore", []string{"view", "export"}, controllers.RecordsIgnoredUpdateHandler),
			util.NewEntryPoint("GET", "records/{id}/marked", []string{"view", "export"}, controllers.NewRecordSSMHandler(dalClient)),
			util.NewEntryPoint("GET", "call", []string{"view"}, controllers.CallStatsGetHandler),
		},
	}

	err = services.InitTags()
	if err != nil {
		return err
	}

	return services.VisitRecordsServiceInit()
}
