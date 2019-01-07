package qi

import (
	"database/sql"
	"net/http"
	"time"

	"emotibot.com/emotigo/module/admin-api/util"
	"emotibot.com/emotigo/module/qic-api/model/v1"
	"emotibot.com/emotigo/module/qic-api/util/logicaccess"
	"emotibot.com/emotigo/pkg/logger"
)

var (
	// ModuleInfo is needed for module define
	ModuleInfo util.ModuleInfo
	tagDao     TagDao
	sqlConn    *sql.DB
	dbLike     model.DBLike
)

func init() {
	ModuleInfo = util.ModuleInfo{
		ModuleName: "qi",
		EntryPoints: []util.EntryPoint{
			util.NewEntryPoint("POST", "groups", []string{}, handleCreateGroup),
			util.NewEntryPoint("GET", "groups", []string{}, handleGetGroups),
			util.NewEntryPoint("GET", "groups/filters", []string{}, handleGetGroupsByFilter),
			util.NewEntryPoint("GET", "groups/{id}", []string{}, handleGetGroup),
			util.NewEntryPoint("PUT", "groups/{id}", []string{}, handleUpdateGroup),
			util.NewEntryPoint("DELETE", "groups/{id}", []string{}, handleDeleteGroup),

			util.NewEntryPoint("GET", "tags", []string{}, HandleGetTags),
			util.NewEntryPoint("POST", "tags", []string{}, HandlePostTags),
			util.NewEntryPoint("PUT", "tags/{tag_id}", []string{}, HandlePutTags),
			util.NewEntryPoint("DELETE", "tags/{tag_id}", []string{}, HandleDeleteTag),

			util.NewEntryPoint("GET", "sentences", []string{}, handleGetSentences),
			util.NewEntryPoint("POST", "sentences", []string{}, handleNewSentence),
			util.NewEntryPoint("GET", "sentences/{id}", []string{}, WithSenUUIDCheck(handleGetSentence)),
			util.NewEntryPoint("PUT", "sentences/{id}", []string{}, WithSenUUIDCheck(handleModifySentence)),
			util.NewEntryPoint("DELETE", "sentences/{id}", []string{}, WithSenUUIDCheck(handleDeleteSentence)),

			util.NewEntryPoint("POST", "sentence-groups", []string{}, handleCreateSentenceGroup),
			util.NewEntryPoint("GET", "sentence-groups", []string{}, handleGetSentenceGroups),
			util.NewEntryPoint("GET", "sentence-groups/{id}", []string{}, handleGetSentenceGroup),
			util.NewEntryPoint("PUT", "sentence-groups/{id}", []string{}, handleUpdateSentenceGroup),
			util.NewEntryPoint("DELETE", "sentence-groups/{id}", []string{}, handleDeleteSentenceGroup),

			util.NewEntryPoint("POST", "conversation-flow", []string{}, handleCreateConversationFlow),
			util.NewEntryPoint("GET", "conversation-flow", []string{}, handleGetConversationFlows),
			util.NewEntryPoint("GET", "conversation-flow/{id}", []string{}, handleGetConversationFlow),
			util.NewEntryPoint("PUT", "conversation-flow/{id}", []string{}, handleUpdateConversationFlow),
			util.NewEntryPoint("DELETE", "conversation-flow/{id}", []string{}, handleDeleteConversationFlow),

			util.NewEntryPoint("POST", "rules", []string{}, handleCreateConversationRule),
			util.NewEntryPoint("GET", "rules", []string{}, handleGetConversationRules),
			util.NewEntryPoint("GET", "rules/{id}", []string{}, handleGetConversationRule),
			util.NewEntryPoint("PUT", "rules/{id}", []string{}, handleUpdateConversationRule),
			util.NewEntryPoint("DELETE", "rules/{id}", []string{}, handleDeleteConversationRule),

			util.NewEntryPoint(http.MethodGet, "calls", []string{}, CallsHandler),
			util.NewEntryPoint(http.MethodPost, "calls", []string{}, NewCallsHandler),
			util.NewEntryPoint(http.MethodPost, "calls/{call_id}/file", []string{}, UpdateCallsFileHandler),
			util.NewEntryPoint(http.MethodGet, "calls/{call_id}/file", []string{}, CallsFileHandler),
		},
		OneTimeFunc: map[string]func(){
			"init db": func() {
				envs := ModuleInfo.Environments

				url := envs["MYSQL_URL"]
				user := envs["MYSQL_USER"]
				pass := envs["MYSQL_PASS"]
				db := envs["MYSQL_DB"]

				newConn, err := util.InitDB(url, user, pass, db)
				sqlConn = newConn
				if err != nil {
					logger.Error.Printf("Cannot init qi db, [%s:%s@%s:%s]: %s\n", user, pass, url, db, err.Error())
					return
				}

				dbLike = &model.DefaultDBLike{
					DB: sqlConn,
				}
				serviceDAO = model.NewGroupSQLDao(sqlConn)
				tagDao, err = model.NewTagSQLDao(sqlConn)
				if err != nil {
					logger.Error.Printf("init tag dao failed, %v", err)
					return
				}
				sentenceDao = model.NewSentenceSQLDao(sqlConn)

				cuURL := envs["LOGIC_PREDICT_URL"]
				predictor = &logicaccess.Client{URL: cuURL, Timeout: time.Duration(3 * time.Second)}
			},
		},
	}
}
