package BF

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"emotibot.com/emotigo/module/admin-api/auth"
	"emotibot.com/emotigo/module/admin-api/util"
	"emotibot.com/emotigo/module/admin-api/util/AdminErrors"
	"emotibot.com/emotigo/module/admin-api/util/requestheader"
	"emotibot.com/emotigo/pkg/logger"
)

var (
	// ModuleInfo is needed for module define
	ModuleInfo util.ModuleInfo
)

const (
	accessTokenExpire = 86400
)

// This module will do some dirty work for BF compatible...
func init() {
	ModuleInfo = util.ModuleInfo{
		ModuleName: "bf",
		EntryPoints: []util.EntryPoint{
			// id must same with token-auth
			// id, name
			util.NewEntryPoint("POST", "enterprise", []string{}, handleAddEnterprise),
			// id, name
			util.NewEntryPoint("PATCH", "enterprise/{id}", []string{}, handleUpdateEnterprise),
			// id
			util.NewEntryPoint("DELETE", "enterprise/{id}", []string{}, handleDeleteEnterprise),

			// id must same with token-auth
			// appid 	userid 	name
			util.NewEntryPoint("POST", "app", []string{}, handleAddApp),
			// appid, name
			util.NewEntryPoint("PATCH", "app/{id}", []string{}, handleUpdateApp),
			// appid
			util.NewEntryPoint("DELETE", "app/{id}", []string{}, handleDeleteApp),

			// id must same with token-auth
			// userid account password enterprise
			util.NewEntryPoint("POST", "user", []string{}, handleAddUser),
			// role
			util.NewEntryPoint("PUT", "user/{id}", []string{}, handleUpdateUserPassword),
			// role
			util.NewEntryPoint("PUT", "user/{id}/role", []string{}, handleUpdateUserRole),
			// userid
			util.NewEntryPoint("DELETE", "user/{id}", []string{}, handleDeleteUser),

			// use name as uuid of role in token-auth
			// id commands
			util.NewEntryPoint("POST", "role", []string{}, handleAddRole),
			// id commands
			util.NewEntryPoint("PUT", "role/{id}", []string{}, handleUpdateRole),
			// id
			util.NewEntryPoint("DELETE", "role/{id}", []string{}, handleDeleteRole),

			// appid
			util.NewEntryPoint("POST", "ssm-data", []string{}, handleInitSSM),

			// using label in ssm
			// util.NewEntryPoint("GET", "labels", []string{"view"}, handleGetLabels),
			// util.NewEntryPoint("PUT", "label/{id}", []string{"view"}, handleUpdateLabel),
			// util.NewEntryPoint("POST", "label", []string{"view"}, handleAddLabel),
			// util.NewEntryPoint("DELETE", "label/{id}", []string{"view"}, handleDeleteLabel),

			util.NewEntryPoint("GET", "cmds", []string{"view"}, handleGetCmds),
			util.NewEntryPoint("GET", "cmd/{id}", []string{"edit"}, handleGetCmd),
			util.NewEntryPoint("PUT", "cmd/{id}", []string{"edit"}, handleUpdateCmd),
			util.NewEntryPoint("PUT", "cmd/{id}/move", []string{"edit"}, handleMoveCmd),
			util.NewEntryPoint("GET", "cmd-class/{id}", []string{"view"}, handleGetCmdClass),

			util.NewEntryPoint("POST", "cmd", []string{"create"}, handleAddCmd),
			util.NewEntryPoint("DELETE", "cmd/{id}", []string{"view"}, handleDeleteCmd),
			util.NewEntryPoint("POST", "cmd-class", []string{"view"}, handleAddCmdClass),
			util.NewEntryPoint("PUT", "cmd-class/{id}", []string{"edit"}, handleUpdateCmdClass),
			util.NewEntryPoint("DELETE", "cmd-class/{id}", []string{"delete"}, handleDeleteCmdClass),

			util.NewEntryPoint("POST", "cmds/import", []string{"edit"}, handleImportCmds),
			util.NewEntryPoint("GET", "cmds/import/{id}", []string{"view"}, handleGetImportCmdsStatus),
			util.NewEntryPoint("GET", "cmds/report/{id}", []string{"view"}, handleGetImportCmdsReport),
			util.NewEntryPoint("GET", "cmds/export", []string{"export"}, handleExportCmds),

			util.NewEntryPoint("GET", "ssm/categories", []string{"view"}, handleGetSSMCategories),
			util.NewEntryPoint("GET", "ssm/labels", []string{"veiw"}, handleGetSSMLabels),
			util.NewEntryPointWithConfig("GET", "access-token", []string{"view"}, handleGetBFAccessToken, util.EntryConfig{
				Version:         1,
				IgnoreAppID:     true,
				IgnoreAuthToken: false,
			}),

			util.NewEntryPointWithVer("GET", "ssm/categories", []string{"view"}, handleGetSSMCategoriesV2, 2),
		},
		EntryPrefix: []util.EntryPoint{
			util.NewEntryPoint("GET", "dal/", []string{}, handleRedirect),
			util.NewEntryPoint("POST", "dal/", []string{}, handleRedirect),
			util.NewEntryPoint("GET", "upload/", []string{}, handleRedirect),
			util.NewEntryPoint("POST", "upload/", []string{}, handleRedirect),
			util.NewEntryPoint("GET", "sq/", []string{}, handleRedirect),
			util.NewEntryPoint("POST", "sq/", []string{}, handleRedirect),
			util.NewEntryPoint("GET", "dimension/", []string{}, handleRedirect),
			util.NewEntryPoint("POST", "dimension/", []string{}, handleRedirect),
			util.NewEntryPoint("GET", "tag/", []string{}, handleRedirect),
			util.NewEntryPoint("POST", "tag/", []string{}, handleRedirect),
			util.NewEntryPoint("GET", "sq_category/", []string{}, handleRedirect),
			util.NewEntryPoint("POST", "sq_category/", []string{}, handleRedirect),
			util.NewEntryPoint("POST", "solution_stage_status/", []string{}, handleRedirect),
			util.NewEntryPoint("POST", "ai/", []string{}, handleRedirect),
		},
	}
}
func handleAddEnterprise(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	name := r.FormValue("name")

	err := addEnterprise(id, name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleUpdateEnterprise(w http.ResponseWriter, r *http.Request) {
	id := util.GetMuxVar(r, "id")
	name := r.FormValue("name")

	err := updateEnterprise(id, name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleDeleteEnterprise(w http.ResponseWriter, r *http.Request) {
	id := util.GetMuxVar(r, "id")

	err := deleteEnterprise(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleAddApp(w http.ResponseWriter, r *http.Request) {
	appid := r.FormValue("appid")
	userid := r.FormValue("userid")
	name := r.FormValue("name")

	err := addApp(appid, userid, name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	appid := util.GetMuxVar(r, "id")
	name := r.FormValue("name")

	err := updateApp(appid, name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	appid := util.GetMuxVar(r, "id")

	err := deleteApp(appid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleAddUser(w http.ResponseWriter, r *http.Request) {
	userid := r.FormValue("id")
	account := r.FormValue("account")
	password := r.FormValue("password")
	enterprise := r.FormValue("enterprise")

	err := addUser(userid, account, password, enterprise)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	userid := util.GetMuxVar(r, "id")
	err := deleteUser(userid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleAddRole(w http.ResponseWriter, r *http.Request) {
	uuid := r.FormValue("id")
	commandStr := strings.TrimSpace(r.FormValue("commands"))

	commands := []string{}
	if commandStr != "" {
		commands = strings.Split(commandStr, ",")
	}

	err := addRole(uuid, commands)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleUpdateRole(w http.ResponseWriter, r *http.Request) {
	uuid := util.GetMuxVar(r, "id")
	commandStr := strings.TrimSpace(r.FormValue("commands"))

	commands := []string{}
	if commandStr != "" {
		commands = strings.Split(commandStr, ",")
	}

	err := updateRole(uuid, commands)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleDeleteRole(w http.ResponseWriter, r *http.Request) {
	uuid := util.GetMuxVar(r, "id")
	if strings.TrimSpace(uuid) == "" {
		return
	}

	err := deleteRole(uuid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	userid := util.GetMuxVar(r, "id")
	roleid := r.FormValue("role")
	enterprise := r.FormValue("enterprise")

	err := updateUserRole(enterprise, userid, roleid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleUpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	userid := util.GetMuxVar(r, "id")
	password := r.FormValue("password")
	enterprise := r.FormValue("enterprise")

	err := updateUserPassword(enterprise, userid, password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func handleInitSSM(w http.ResponseWriter, r *http.Request) {
	appid := r.FormValue("appid")
	envs := util.GetEnvOf(ModuleInfo.ModuleName)
	url := envs["DAL_URL"]
	if url == "" {
		url = "http://172.17.0.1:8885/dal"
	}

	options := map[string]interface{}{
		"op":       "insert",
		"category": "app",
		"appid":    appid,
	}

	ret, err := util.HTTPPostJSON(url, options, 30)
	if err != nil {
		w.Write([]byte(err.Error()))
	} else {
		w.Write([]byte(ret))
	}
}

func handleGetSSMCategories(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	appid := requestheader.GetAppID(r)
	retObj, err = GetSSMCategories(appid)
	if err != nil {
		util.ReturnError(w, AdminErrors.ErrnoDBError, err.Error())
	} else {
		util.Return(w, nil, retObj)
	}
}

func handleGetSSMCategoriesV2(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	appid := requestheader.GetAppID(r)
	retObj, err = GetSSMCategoriesV2(appid)
	if err != nil {
		util.ReturnError(w, AdminErrors.ErrnoDBError, err.Error())
	} else {
		util.Return(w, nil, retObj)
	}
}

func handleGetSSMLabels(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	appid := requestheader.GetAppID(r)
	retObj, err = GetSSMLabels(appid)
	if err != nil {
		util.ReturnError(w, AdminErrors.ErrnoDBError, err.Error())
	} else {
		util.Return(w, nil, retObj)
	}
}

func handleGetBFAccessToken(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	userid := requestheader.GetUserID(r)
	retObj, err = GetBFAccessToken(userid)
	if err != nil {
		util.ReturnError(w, AdminErrors.ErrnoDBError, err.Error())
	} else {
		util.Return(w, nil, retObj)
	}
}

var tokenCache = map[string]string{}
var tokenExpireTime = map[string]int64{}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	var resp *http.Response
	var err error
	var req *http.Request
	client := &http.Client{}
	bfServer := ModuleInfo.Environments["SERVER_URL"]
	now := time.Now()

	token := requestheader.GetAuthToken(r)
	params := strings.Split(token, " ")
	if len(params) != 2 {
		logger.Trace.Println("Error token format:", token)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Find from cache
	accessToken := ""
	if val, ok := tokenCache[params[1]]; ok {
		if now.Unix() <= tokenExpireTime[params[1]] {
			accessToken = val
		}
	}

	// If not in cache, gen new token
	if accessToken == "" {
		switch params[0] {
		case "Bearer":
			accessToken, err = GetBFAccessToken(requestheader.GetUserID(r))
		case "Api":
			// Api type will gen new access token of enterprise admin
			_, appid, err := auth.GetAppOwner(params[1])
			if err == nil {
				accessToken, err = GetNewAccessTokenOfAppid(appid)
			}
			if appid == "" {
				err = errors.New("Redirect only support apiKey of robot")
			}
		}
	}

	// if gen token fail, return error
	if err != nil {
		logger.Trace.Println("Gen access token fail:", err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if accessToken == "" {
		logger.Trace.Println("Get empty access token")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokenCache[params[1]] = accessToken
	tokenExpireTime[params[1]] = now.Unix() + accessTokenExpire

	realPath := strings.Replace(r.RequestURI, "api/v1/bf/", "", -1)
	url := fmt.Sprintf("http://%s%s", bfServer, realPath)
	logger.Trace.Printf("%v %v", r.Method, url)
	req, err = http.NewRequest(r.Method, url, r.Body)
	for name, value := range r.Header {
		req.Header.Set(name, value[0])
	}
	req.Header.Set("Access_token", accessToken)
	resp, err = client.Do(req)
	r.Body.Close()

	// combined for GET/POST
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range resp.Header {
		w.Header().Set(k, v[0])
	}

	logger.Trace.Printf("Redirect headers: %+v\n", w.Header())

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	resp.Body.Close()

}
