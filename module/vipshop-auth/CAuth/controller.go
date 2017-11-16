package CAuth

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"emotibot.com/emotigo/module/vipshop-admin/ApiError"
	"emotibot.com/emotigo/module/vipshop-admin/util"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

var (
	// ModuleInfo is needed for module define
	ModuleInfo util.ModuleInfo
)

const validAppID = "vipshop"

func init() {
	ModuleInfo = util.ModuleInfo{
		ModuleName: "cauth",
		EntryPoints: []util.EntryPoint{
			// Privileges and users is readonly in VIP's CAuth system
			util.NewEntryPoint("GET", "privileges", []string{}, handleListPrivilege),
			util.NewEntryPoint("GET", "users", []string{}, handleUserList),
			util.NewEntryPoint("GET", "roles", []string{}, handleRoleList),

			util.NewEntryPoint("PATCH", "user/{id:string}", []string{}, handleUserUpdate),
			util.NewEntryPoint("PATCH", "role/{id:string}", []string{}, handleRoleUpdate),

			util.NewEntryPoint("POST", "role/register", []string{}, handleAddRole),

			util.NewEntryPoint("DELETE", "role/{id:string}", []string{}, handleDeleteRole),
		},
	}
}

func handleListPrivilege(ctx context.Context) {
	appid := util.GetAppID(ctx)
	if appid != validAppID {
		ctx.StatusCode(iris.StatusUnauthorized)
		return
	}

	ret := []*Privilege{}
	for _, priv := range PrivilegesMap {
		ret = append(ret, priv)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].ID < ret[j].ID
	})

	ctx.JSON(GenRetObj(ApiError.SUCCESS, ret))
}

func handleRoleList(ctx context.Context) {
	appid := util.GetAppID(ctx)
	if appid != validAppID {
		ctx.StatusCode(iris.StatusUnauthorized)
		return
	}

	CAuthRoles, err := getRolesFromCAuth()
	if err != nil {
		ctx.JSON(GenRetObj(ApiError.WEB_REQUEST_ERROR, err))
		return
	}

	ret := []*Role{}
	for _, role := range CAuthRoles.Data {
		temp := &Role{
			RoleID:   role.RoleName,
			RoleName: role.RoleName,
		}

		privList, err := getRolePrivs(role.RoleName)
		if err != nil {
			ctx.JSON(GenRetObj(ApiError.WEB_REQUEST_ERROR, err.Error()))
			return
		}
		privStr, _ := json.Marshal(privList)
		temp.Privilege = string(privStr)

		ret = append(ret, temp)
	}

	ctx.JSON(GenRetObj(ApiError.SUCCESS, ret))
}

func handleUserList(ctx context.Context) {
	appid := util.GetAppID(ctx)
	if appid != validAppID {
		ctx.StatusCode(iris.StatusUnauthorized)
		return
	}

	CAuthRoles, err := getRolesFromCAuth()
	if err != nil {
		ctx.JSON(GenRetObj(ApiError.WEB_REQUEST_ERROR, err.Error()))
		return
	}

	userIDList := []string{}
	users := []*UserProp{}
	for _, role := range CAuthRoles.Data {
		CAuthUsers, err := getUsersOfRoleFromCAuth(role.RoleName)
		if err != nil {
			ctx.JSON(GenRetObj(ApiError.WEB_REQUEST_ERROR, err.Error()))
			return
		}
		for _, CAuthUser := range CAuthUsers.Data {
			if util.Contains(userIDList, CAuthUser.UserAcountID) {
				continue
			}
			fmt.Printf("Get user [%s]:%s\n", role.RoleName, CAuthUser.UserName)
			user := &UserProp{
				UserId:   CAuthUser.UserAcountID,
				UserName: CAuthUser.UserName,
				UserType: 1,
				RoleId:   role.RoleName,
			}
			users = append(users, user)
			userIDList = append(userIDList, CAuthUser.UserAcountID)
		}
	}

	ctx.JSON(GenRetObj(ApiError.SUCCESS, users))
}

func handleUserUpdate(ctx context.Context) {
	id := ctx.Params().GetEscape("id")
	appid := util.GetAppID(ctx)
	if appid != validAppID {
		ctx.StatusCode(iris.StatusUnauthorized)
		return
	}
	operator := util.GetUserID(ctx)
	userIP := util.GetUserIP(ctx)
	result := 0

	if len(strings.Trim(id, " ")) == 0 {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(util.GenSimpleRetObj(ApiError.REQUEST_ERROR))
		return
	}

	roleID := strings.Trim(ctx.FormValue("role_id"), " ")

	origUserRoles, err := getUserRoles(id)
	if err != nil {
		util.LogTrace.Printf("Cannot get orig role of user, %s", err.Error())
		ctx.JSON(util.GenRetObj(ApiError.WEB_REQUEST_ERROR, err.Error()))
		return
	}

	origRoles := []string{}
	for _, role := range origUserRoles {
		origRoles = append(origRoles, role.RoleName)
	}

	logMsg := fmt.Sprintf("Update user (%s) role: [%s] -> [%s]", id, strings.Join(origRoles, ","), roleID)

	err = updateUserRole(operator, id, origUserRoles, roleID)
	if err != nil {
		ctx.JSON(util.GenRetObj(ApiError.WEB_REQUEST_ERROR, err.Error()))
	} else {
		ctx.JSON(util.GenSimpleRetObj(ApiError.SUCCESS))
		result = 1
	}
	util.AddAuditLog(operator, userIP, util.AuditModuleMembers, util.AuditOperationEdit, logMsg, result)
}

func handleRoleUpdate(ctx context.Context) {
	id := ctx.Params().GetEscape("id")
	appid := util.GetAppID(ctx)
	if appid != validAppID {
		ctx.StatusCode(iris.StatusUnauthorized)
		return
	}
	operator := util.GetUserID(ctx)
	userIP := util.GetUserIP(ctx)
	result := 0

	if len(strings.Trim(id, " ")) == 0 {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(util.GenSimpleRetObj(ApiError.REQUEST_ERROR))
		return
	}

	newPrivStr := ctx.FormValue("privilege")
	newRolePriv := make(map[int][]string)
	err := json.Unmarshal([]byte(newPrivStr), &newRolePriv)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(util.GenRetObj(ApiError.REQUEST_ERROR, err.Error()))
		return
	}

	origRolePriv, err := getRolePrivs(id)
	if err != nil {
		ctx.JSON(util.GenRetObj(ApiError.WEB_REQUEST_ERROR, err.Error()))
		return
	}

	origPrivStr, _ := json.Marshal(origRolePriv)
	// newPrivStr, _ := json.Marshal(newRolePriv)
	logMsg := fmt.Sprintf("Update role (%s) priv: [%s] -> [%s]", id, origPrivStr, newPrivStr)

	err = updateRolePriv(operator, id, origRolePriv, newRolePriv)
	if err != nil {
		ctx.JSON(util.GenRetObj(ApiError.WEB_REQUEST_ERROR, err.Error()))
	} else {
		ret := Role{
			Privilege: newPrivStr,
			RoleID:    id,
			RoleName:  id,
		}
		ctx.JSON(util.GenRetObj(ApiError.SUCCESS, ret))
		result = 1
	}
	util.AddAuditLog(operator, userIP, util.AuditModuleMembers, util.AuditOperationEdit, logMsg, result)
}

func handleAddRole(ctx context.Context) {
	userID := util.GetUserID(ctx)
	userIP := util.GetUserIP(ctx)
	roleName := strings.Trim(ctx.FormValue("role_name"), " ")
	result := 0

	err := addRole(roleName)
	if err != nil {
		ctx.JSON(util.GenRetObj(ApiError.WEB_REQUEST_ERROR, err.Error()))
	} else {
		ret := Role{
			Privilege: "{}",
			RoleID:    roleName,
			RoleName:  roleName,
		}
		ctx.JSON(util.GenRetObj(ApiError.SUCCESS, ret))
		result = 1
	}
	logMsg := fmt.Sprintf("Add new role: %s", roleName)
	util.AddAuditLog(userID, userIP, util.AuditModuleRole, util.AuditOperationAdd, logMsg, result)
}

func handleDeleteRole(ctx context.Context) {
	userID := util.GetUserID(ctx)
	userIP := util.GetUserIP(ctx)
	id := ctx.Params().GetEscape("id")
	result := 0

	err := deleteRole(id)
	if err != nil {
		ctx.JSON(util.GenRetObj(ApiError.WEB_REQUEST_ERROR, err.Error()))
	} else {
		ctx.JSON(util.GenSimpleRetObj(ApiError.SUCCESS))
		result = 1
	}
	logMsg := fmt.Sprintf("delete role: %s", id)
	util.AddAuditLog(userID, userIP, util.AuditModuleRole, util.AuditOperationAdd, logMsg, result)
}
