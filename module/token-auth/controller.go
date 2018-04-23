package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"emotibot.com/emotigo/module/token-auth/data"
	"emotibot.com/emotigo/module/token-auth/enum"
	"emotibot.com/emotigo/module/token-auth/util"
	"github.com/gorilla/mux"
)

func EnterprisesGetHandler(w http.ResponseWriter, r *http.Request) {
	retData, errMsg := getEnterprises()
	returnOKMsg(w, errMsg, retData)
}

func EnterpriseGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}

	retData, err := getEnterprise(enterpriseID)
	if err != nil {
		returnMsg(w, err.Error(), retData)
	} else {
		returnMsg(w, "", retData)
	}
}

func UsersGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}

	retData, err := getUsers(enterpriseID)
	if err != nil {
		returnInternalError(w, err.Error())
	} else {
		returnSuccess(w, retData)
	}
}

func UserGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}

	userID := vars["userID"]
	if !util.IsValidUUID(userID) {
		returnBadRequest(w, "userID")
		return
	}

	retData, err := getUser(enterpriseID, userID)
	if err != nil {
		returnInternalError(w, err.Error())
	} else {
		returnSuccess(w, retData)
	}
}

func AppsGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}

	retData, errMsg := getApps(enterpriseID)
	returnMsg(w, errMsg, retData)
}

func AppGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}

	appID := vars["appID"]
	if !util.IsValidUUID(appID) {
		returnBadRequest(w, "appID")
		return
	}
	retData, errMsg := getApp(enterpriseID, appID)
	returnMsg(w, errMsg, retData)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	account := r.Form.Get("account")
	passwd := r.Form.Get("passwd")
	if !util.IsValidString(&passwd) || !util.IsValidString(&account) {
		returnBadRequest(w, "")
		return
	}

	enterprise, user, errMsg := login(account, passwd)
	if errMsg != "" {
		returnInternalError(w, errMsg)
		return
	} else if enterprise == nil && user == nil {
		returnForbidden(w)
		return
	}

	token, err := user.GenerateToken()
	if err != nil {
		returnInternalError(w, err.Error())
		return
	}

	loginRet := data.LoginInfo{
		Token: token,
		Info:  user,
	}
	returnOKMsg(w, errMsg, loginRet)

}

func UserAddHandler(w http.ResponseWriter, r *http.Request) {
	requester := getRequester(r)
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}
	user, err := parseAddUserFromRequest(r)
	if err != nil {
		returnBadRequest(w, err.Error())
		return
	}
	roleID := r.FormValue("role")
	if roleID == "" && user.Type == 2 {
		returnBadRequest(w, "Role")
		return
	}

	if requester.Type > user.Type {
		returnForbidden(w)
		return
	}

	if roleID != "" {
		role, err := getRole(enterpriseID, roleID)
		if err != nil && err != sql.ErrNoRows {
			util.LogError.Printf("Error when get role %s: %s\n", roleID, err.Error())
			returnInternalError(w, err.Error())
			return
		} else if role == nil {
			returnBadRequest(w, "Role")
		}
	}

	id, err := addUser(enterpriseID, user, roleID)
	if err != nil {
		returnInternalError(w, err.Error())
		return
	}
	newUser, err := getUser(enterpriseID, id)
	if err != nil {
		returnInternalError(w, err.Error())
		return
	}
	returnSuccess(w, newUser)
}

func UserDeleteHandler(w http.ResponseWriter, r *http.Request) {
	requester := getRequester(r)
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}
	userID := vars["userID"]
	if !util.IsValidUUID(userID) {
		returnBadRequest(w, "userID")
		return
	}

	user, err := getUser(enterpriseID, userID)
	if err != nil && err != sql.ErrNoRows {
		returnInternalError(w, err.Error())
		return
	} else if user == nil {
		returnSuccess(w, "")
		return
	}

	if requester.Type > user.Type {
		returnForbidden(w)
		return
	}

	err = deleteUser(enterpriseID, userID)
	if err != nil {
		returnInternalError(w, err.Error())
		return
	}

	returnSuccess(w, "")
}

func UserUpdateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}

	userID := vars["userID"]
	if !util.IsValidUUID(userID) {
		returnBadRequest(w, "userID")
		return
	}

	origUser, err := getUser(enterpriseID, userID)
	if err != nil {
		returnInternalError(w, err.Error())
		return
	} else if origUser == nil {
		returnNotFound(w)
		return
	}

	newUser, err := parseUpdateUserFromRequest(r)
	if err != nil {
		returnBadRequest(w, err.Error())
		return
	}
	newUser.Type = origUser.Type

	newUser.ID = userID
	newUser.Enterprise = &enterpriseID
	err = updateUser(enterpriseID, newUser)
	if err != nil {
		returnInternalError(w, err.Error())
		return
	}

	updatedUser, err := getUser(enterpriseID, userID)
	if err != nil {
		returnInternalError(w, err.Error())
		return
	}
	returnSuccess(w, updatedUser)
}

func RolesGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}

	retData, err := getRoles(enterpriseID)
	if err != nil {
		returnInternalError(w, err.Error())
	} else {
		returnSuccess(w, retData)
	}
}
func RoleGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}
	roleID := vars["roleID"]
	if !util.IsValidUUID(roleID) {
		returnBadRequest(w, "roleID")
		return
	}

	retData, err := getRole(enterpriseID, roleID)
	if err != nil {
		returnInternalError(w, err.Error())
	} else {
		if retData == nil {
			returnNotFound(w)
		} else {
			returnSuccess(w, retData)
		}
	}
}
func RoleDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}
	roleID := vars["roleID"]
	if !util.IsValidUUID(roleID) {
		returnBadRequest(w, "roleID")
		return
	}

	retData, err := deleteRole(enterpriseID, roleID)
	if err != nil {
		returnInternalError(w, err.Error())
	} else {
		returnSuccess(w, retData)
	}
}
func RoleAddHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}
	role, err := parseRoleFromRequest(r)
	if err != nil {
		returnBadRequest(w, err.Error())
		return
	}
	id, err := addRole(enterpriseID, role)
	if err != nil {
		returnInternalError(w, err.Error())
		return
	}
	newRole, err := getRole(enterpriseID, id)
	if err != nil {
		returnInternalError(w, err.Error())
		return
	}
	returnSuccess(w, newRole)
}
func RoleUpdateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}
	roleID := vars["roleID"]
	if !util.IsValidUUID(roleID) {
		returnBadRequest(w, "roleID")
		return
	}
	role, err := parseRoleFromRequest(r)
	if err != nil {
		returnBadRequest(w, err.Error())
		return
	}
	ret, err := updateRole(enterpriseID, roleID, role)
	if err != nil {
		if err == sql.ErrNoRows {
			returnNotFound(w)
			return
		}
		returnInternalError(w, err.Error())
		return
	}
	returnSuccess(w, ret)
}

func ModulesGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enterpriseID := vars["enterpriseID"]
	if !util.IsValidUUID(enterpriseID) {
		returnBadRequest(w, "enterpriseID")
		return
	}

	retData, err := getModules(enterpriseID)
	if err != nil {
		returnInternalError(w, err.Error())
	} else {
		returnSuccess(w, retData)
	}
}

func ValidateTokenHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	if token == "" {
		params := strings.Split(r.Header.Get("Authorization"), " ")
		if len(params) >= 2 {
			token = params[1]
		}
	}
	if token == "" {
		returnBadRequest(w, "token")
		return
	}

	userInfo := data.User{}
	err := userInfo.SetValueWithToken(token)
	if err != nil {
		util.LogInfo.Println("Check token fail: ", err.Error())
		returnBadRequest(w, "token")
		return
	}
	returnSuccess(w, nil)
}

func parseRoleFromRequest(r *http.Request) (*data.Role, error) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		return nil, errors.New("Invalid name")
	}
	discription := r.FormValue("description")
	privilegeStr := r.FormValue("privilege")

	privileges := map[string][]string{}
	err := json.Unmarshal([]byte(privilegeStr), &privileges)
	if err != nil {
		util.LogError.Printf("Cannot decode privilegeStr: %s\n", err.Error())
		return nil, err
	}
	ret := data.Role{
		Name:        name,
		Discription: discription,
		Privileges:  privileges,
	}
	return &ret, nil
}

func loadUserFromRequest(r *http.Request) *data.User {
	user := data.User{}
	username := r.FormValue("username")
	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	role := r.FormValue("role")
	user.Email = &email
	user.DisplayName = &name
	user.Password = &password
	user.Role = &role
	user.UserName = &username

	userType, err := strconv.Atoi(r.FormValue("type"))
	if err != nil {
		userType = enum.NormalUser
	} else if userType > enum.NormalUser || userType < enum.AdminUser {
		userType = enum.NormalUser
	}
	user.Type = userType

	return &user
}
func parseAddUserFromRequest(r *http.Request) (*data.User, error) {
	user := loadUserFromRequest(r)

	// if user.Email == nil || *user.Email == "" {
	// 	return nil, errors.New("invalid email")
	// }
	if user.Password == nil || *user.Password == "" {
		return nil, errors.New("invalid password")
	}
	if user.UserName == nil || *user.UserName == "" {
		return nil, errors.New("invalid username")
	}

	return user, nil
}
func parseUpdateUserFromRequest(r *http.Request) (*data.User, error) {
	user := loadUserFromRequest(r)

	// if user.Email == nil || *user.Email == "" {
	// 	return nil, errors.New("invalid email")
	// }

	return user, nil
}

func returnMsg(w http.ResponseWriter, errMsg string, retData interface{}) {
	if reflect.ValueOf(retData).IsNil() && errMsg == "" {
		returnNotFound(w)
	} else {
		returnOKMsg(w, errMsg, retData)
	}
}

func returnOKMsg(w http.ResponseWriter, errMsg string, retData interface{}) {
	if errMsg != "" {
		writeErrJSON(w, errMsg)
	} else {
		returnSuccess(w, retData)
	}
}

func returnNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	writeErrJSON(w, "Resource not found")
}

func returnBadRequest(w http.ResponseWriter, column string) {
	errMsg := ""
	w.WriteHeader(http.StatusBadRequest)
	if column != "" {
		errMsg = fmt.Sprintf("Column input error: %s", column)
	} else {
		errMsg = "Bad request"
	}
	writeErrJSON(w, errMsg)
}

func returnUnauthorized(w http.ResponseWriter) {
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func returnForbidden(w http.ResponseWriter) {
	http.Error(w, "Forbidden", http.StatusForbidden)
}

func returnInternalError(w http.ResponseWriter, errMsg string) {
	w.WriteHeader(http.StatusInternalServerError)
	writeErrJSON(w, errMsg)
}

func returnSuccess(w http.ResponseWriter, retData interface{}) {
	ret := data.Return{
		ReturnMessage: "success",
		ReturnObj:     &retData,
	}

	writeResponseJSON(w, &ret)
}

func writeErrJSON(w http.ResponseWriter, errMsg string) {
	ret := data.Return{
		ReturnMessage: errMsg,
		ReturnObj:     nil,
	}
	writeResponseJSON(w, &ret)
}

func writeResponseJSON(w http.ResponseWriter, ret *data.Return) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&ret)
}