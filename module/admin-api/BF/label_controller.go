package BF

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"emotibot.com/emotigo/module/admin-api/ApiError"
	"emotibot.com/emotigo/module/admin-api/util"
	"emotibot.com/emotigo/module/admin-api/util/requestheader"
	"emotibot.com/emotigo/pkg/logger"
	"emotibot.com/emotigo/module/admin-api/util/audit"
	"emotibot.com/emotigo/module/admin-api/util/AdminErrors"
	"bytes"
	"io"
	"emotibot.com/emotigo/module/admin-api/util/localemsg"
	"emotibot.com/emotigo/module/admin-api/util/validate"
)

func handleGetCmds(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	retCode := ApiError.SUCCESS
	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(retCode, retObj))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(retCode, err.Error()), ApiError.GetHttpStatus(retCode))
		}
	}()
	appid := requestheader.GetAppID(r)

	cmds, err := GetCmds(appid)
	if err != nil {
		retCode = ApiError.DB_ERROR
	} else {
		retObj = cmds
	}
}
func handleGetCmd(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	retCode := ApiError.SUCCESS
	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(retCode, retObj))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(retCode, err.Error()), ApiError.GetHttpStatus(retCode))
		}
	}()
	appid := requestheader.GetAppID(r)
	id, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		retCode = ApiError.REQUEST_ERROR
		retObj = err.Error()
		return
	}

	cmd, err := GetCmd(appid, id)
	if cmd == nil {
		retCode = ApiError.NOT_FOUND_ERROR
		err = util.ErrNotFound
		return
	} else if err != nil {
		retCode = ApiError.DB_ERROR
		return
	}

	retObj = cmd
	return
}
func handleUpdateCmd(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	appid := requestheader.GetAppID(r)
	retCode := ApiError.SUCCESS
	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(retCode, retObj))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(retCode, err.Error()), ApiError.GetHttpStatus(retCode))
		}
	}()

	id, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		retCode = ApiError.REQUEST_ERROR
		err = util.GenBadRequestError("ID")
		return
	}

	origCmd, err := GetCmd(appid, id)
	if origCmd == nil {
		retCode = ApiError.NOT_FOUND_ERROR
		err = util.ErrNotFound
		return
	}
	if err != nil {
		retCode = ApiError.DB_ERROR
		return
	}

	cmd, err := parseCmdFromRequest(r)
	if err != nil {
		retCode = ApiError.REQUEST_ERROR
		return
	}

	retCode, err = UpdateCmd(appid, id, cmd)
	if err != nil {
		return
	}
	cmd.ID = id
	retObj = cmd
	go util.ConsulUpdateCmd(appid)
}
func handleAddCmd(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	appid := requestheader.GetAppID(r)
	retCode := ApiError.SUCCESS
	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(retCode, retObj))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(retCode, err.Error()), ApiError.GetHttpStatus(retCode))
		}
	}()

	cmd, err := parseCmdFromRequest(r)
	if err != nil {
		retCode = ApiError.REQUEST_ERROR
		return
	}
	cid, err := strconv.Atoi(r.FormValue("cid"))
	if err != nil {
		err = util.GenBadRequestError(util.Msg["CmdParentID"])
		retCode = ApiError.REQUEST_ERROR
		return
	}

	id, retCode, err := AddCmd(appid, cmd, cid)
	if err != nil {
		return
	}
	cmd.ID = id
	retObj = cmd
	go util.ConsulUpdateCmd(appid)
}
func handleDeleteCmd(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	appid := requestheader.GetAppID(r)
	retCode := ApiError.SUCCESS
	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(retCode, retObj))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(retCode, err.Error()), ApiError.GetHttpStatus(retCode))
		}
	}()
	id, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		retCode = ApiError.REQUEST_ERROR
		retObj = err.Error()
		return
	}

	err = DeleteCmd(appid, id)
	if err != nil {
		retCode = ApiError.DB_ERROR
	}
	if err == nil {
		go util.ConsulUpdateCmd(appid)
	}
}
func handleGetCmdsOfLabel(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	status, retCode := http.StatusOK, ApiError.SUCCESS
	defer func() {
		if status == http.StatusOK {
			util.WriteJSON(w, util.GenRetObj(retCode, retObj))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(retCode, retObj), status)
		}
	}()
	appid := requestheader.GetAppID(r)
	labelID, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		status, retCode = http.StatusBadRequest, ApiError.REQUEST_ERROR
		return
	}

	cmds, err := GetCmdsOfLabel(appid, labelID)
	if err != nil {
		status, retCode = http.StatusInternalServerError, ApiError.DB_ERROR
		retObj = err.Error()
	} else {
		retObj = cmds
	}
}
func parseCmdFromRequest(r *http.Request) (cmd *Cmd, err error) {
	err = r.ParseForm()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			logger.Info.Printf("Parse cmd fail: %s\n", err.Error())
		}
	}()

	ret := Cmd{}
	ret.Name = r.FormValue("name")
	ret.Answer = r.FormValue("answer")
	ret.Status = r.FormValue("status") == "true" ||
		r.FormValue("status") == "T" ||
		r.FormValue("status") == "1"
	begin, err := time.Parse(time.RFC3339, r.FormValue("begin_time"))
	if err != nil {
		ret.Begin = nil
	} else {
		ret.Begin = &begin
	}
	end, err := time.Parse(time.RFC3339, r.FormValue("end_time"))
	if err != nil {
		ret.End = nil
	} else {
		ret.End = &end
	}

	target, err := strconv.Atoi(r.FormValue("target"))
	if err != nil {
		err = errors.New("Invalid target")
		return
	}
	rtype, err := strconv.Atoi(r.FormValue("response_type"))
	if err != nil {
		err = errors.New("Invalid response type")
		return
	}

	if target > ret.Target.Max() || target < 0 {
		err = errors.New("Invalid target")
		return
	}
	if rtype > ret.Type.Max() || rtype < 0 {
		err = errors.New("Invalid response type")
		return
	}
	ret.Target = CmdTarget(target)
	ret.Type = ResponseType(rtype)

	ruleStr := r.FormValue("rule")
	ruleContents := []*CmdContent{}
	err = json.Unmarshal([]byte(ruleStr), &ruleContents)
	if err != nil {
		err = fmt.Errorf("Invalid rule content: %s", err.Error())
		return
	}
	for i, r := range ruleContents {
		if !r.IsValid() {
			err = fmt.Errorf("rule content error of rule %d", i+1)
			return
		}
	}
	ret.Rule = ruleContents

	labelsStr := r.FormValue("labels")
	labelIDs := []int{}
	err = json.Unmarshal([]byte(labelsStr), &labelIDs)
	if err != nil {
		return
	}
	existedLabel := map[string]bool{}
	for _, id := range labelIDs {
		idx := fmt.Sprintf("%d", id)
		if _, ok := existedLabel[idx]; !ok {
			ret.LinkLabel = append(ret.LinkLabel, id)
			existedLabel[idx] = true
		}
	}

	cmd = &ret
	return
}
func handleGetLabelsOfCmd(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	retCode := ApiError.SUCCESS
	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(retCode, retObj))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(retCode, err.Error()), ApiError.GetHttpStatus(retCode))
		}
	}()
	appid := requestheader.GetAppID(r)
	cmdID, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		retCode = ApiError.REQUEST_ERROR
		return
	}

	labels, err := GetLabelsOfCmd(appid, cmdID)
	if err != nil {
		retCode = ApiError.DB_ERROR
	} else {
		retObj = labels
	}
}
func handleGetCmdClass(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	retCode := ApiError.SUCCESS
	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(retCode, retObj))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(retCode, err.Error()), ApiError.GetHttpStatus(retCode))
		}
	}()
	appid := requestheader.GetAppID(r)
	classID, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		retCode, err = ApiError.REQUEST_ERROR, util.GenBadRequestError("ID")
		return
	}

	retObj, retCode, err = GetCmdClass(appid, classID)
	return
}
func handleDeleteCmdClass(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	retCode := ApiError.SUCCESS
	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(retCode, retObj))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(retCode, err.Error()), ApiError.GetHttpStatus(retCode))
		}
	}()
	appid := requestheader.GetAppID(r)
	classID, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		err = util.GenBadRequestError(util.Msg["CmdParentID"])
		retCode = ApiError.REQUEST_ERROR
		return
	}

	err = DeleteCmdClass(appid, classID)
	if err != nil {
		retCode = ApiError.DB_ERROR
		return
	}
}
func handleAddCmdClass(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	retCode := ApiError.SUCCESS
	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(retCode, retObj))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(retCode, err.Error()), ApiError.GetHttpStatus(retCode))
		}
	}()
	appid := requestheader.GetAppID(r)
	className := r.FormValue("name")
	if strings.TrimSpace(className) == "" {
		retCode, err = ApiError.REQUEST_ERROR, util.GenBadRequestError(util.Msg["CmdClassName"])
		return
	}

	// class layer is only one for now
	// pid, err := strconv.Atoi(r.FormValue("pid"))
	// if err != nil {
	// 	retCode, retObj = ApiError.REQUEST_ERROR, fmt.Sprintf("get pid fail: %s", err.Error())
	// 	status = http.StatusBadRequest
	// 	return
	// }
	// class, err = GetCmdClass(appid, pid)
	// if err != nil {
	// 	retCode, retObj = ApiError.DB_ERROR, fmt.Sprintf("get parent class fail")
	// }

	var pid *int
	classID, retCode, err := AddCmdClass(appid, pid, className)
	if err != nil {
		return
	}
	retObj, retCode, err = GetCmdClass(appid, classID)
}
func handleUpdateCmdClass(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	retCode := ApiError.SUCCESS
	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(retCode, retObj))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(retCode, err.Error()), ApiError.GetHttpStatus(retCode))
		}
	}()
	appid := requestheader.GetAppID(r)
	classID, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		err = util.GenBadRequestError(util.Msg["CmdParentID"])
		retCode = ApiError.REQUEST_ERROR
		return
	}

	newClassName := r.FormValue("name")
	if strings.TrimSpace(newClassName) == "" {
		retCode, err = ApiError.REQUEST_ERROR, util.GenBadRequestError(util.Msg["CmdClassName"])
		return
	}

	retCode, err = UpdateCmdClass(appid, classID, newClassName)
	if err != nil {
		return
	}
	retObj, retCode, err = GetCmdClass(appid, classID)
}

func handleMoveCmd(w http.ResponseWriter, r *http.Request) {
	var retObj interface{}
	var err error
	appid := requestheader.GetAppID(r)
	retCode := ApiError.SUCCESS
	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(retCode, retObj))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(retCode, err.Error()), ApiError.GetHttpStatus(retCode))
		}
	}()

	id, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		retCode = ApiError.REQUEST_ERROR
		err = util.GenBadRequestError("ID")
		return
	}
	cid, err := strconv.Atoi(r.FormValue("cid"))
	if err != nil {
		retCode = ApiError.REQUEST_ERROR
		err = util.GenBadRequestError(util.Msg["CmdParentID"])
		return
	}
	if cid != -1 {
		var parentClass *CmdClass
		parentClass, _, err = GetCmdClass(appid, cid)
		if parentClass == nil {
			retCode = ApiError.REQUEST_ERROR
			err = errors.New(util.Msg["ErrorCmdParentNotFound"])
			return
		}
		if err != nil {
			retCode = ApiError.DB_ERROR
			err = errors.New(util.Msg["ErrorCmdParentNotFound"])
			return
		}
	}

	origCmd, err := GetCmd(appid, id)
	if origCmd == nil {
		retCode = ApiError.NOT_FOUND_ERROR
		err = util.ErrNotFound
		return
	}
	if err != nil {
		retCode = ApiError.DB_ERROR
		return
	}

	retCode, err = MoveCmd(appid, id, cid)
	if err != nil {
		return
	}
	go util.ConsulUpdateCmd(appid)
}


func handleImportCmds(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)
	userid := requestheader.GetUserID(r)
	locale := requestheader.GetLocale(r)
	var err AdminErrors.AdminError
	var auditMsg bytes.Buffer
	var recordId int
	var status int

	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(status, recordId))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(status, err.Error()), ApiError.GetHttpStatus(status))
		}

		audit.AddAuditFromRequestAuto(r, auditMsg.String(), status)
		//util.Return(w, err, auditMsg.String())
	}()
	auditMsg.WriteString(util.Msg["UploadBatchCommands"])

	file, info, ioErr := r.FormFile("file")

	if ioErr != nil {
		err = AdminErrors.New(AdminErrors.ErrnoIOError, "")
		return
	}

	var buffer bytes.Buffer
	_, ioErr = io.Copy(&buffer, file)
	if ioErr != nil {
		err = AdminErrors.New(AdminErrors.ErrnoIOError, "")
		return
	}
	auditMsg.WriteString(info.Filename)

	status, recordId, parseErr := ProcessImportCmdFile(appid, userid, buffer.Bytes(), info, locale)

	if parseErr != nil {
		err = AdminErrors.New(AdminErrors.ErrnoRequestError, parseErr.Error())
		return
	}

	go util.ConsulUpdateCmd(appid)
	return
}
func handleGetImportCmdsStatus(w http.ResponseWriter, r *http.Request) {
	recordId, _ := util.GetMuxIntVar(r, "id")

	//rId, _ := strconv.Atoi(recordId)
	var err AdminErrors.AdminError
	var ret int
	defer func() {
		if err == nil {
			util.WriteJSON(w, util.GenRetObj(AdminErrors.ErrnoSuccess, ret))
		} else {
			util.WriteJSONWithStatus(w, util.GenRetObj(err.Errno(), err.Error()), ApiError.GetHttpStatus(err.Errno()))
		}
	}()
	ret, err = GetCmdImportProcess(recordId)
}
func handleGetImportCmdsReport(w http.ResponseWriter, r *http.Request) {

	appid := requestheader.GetAppID(r)
	locale := requestheader.GetLocale(r)
	recordId, _ := util.GetMuxIntVar(r, "id")
	var err AdminErrors.AdminError
	var ret []byte
	var auditMsg bytes.Buffer
	var filename string

	defer func() {
		retVal := 0
		if err == nil {
			retVal = 1
			//now := time.Now()
			//filename := fmt.Sprintf("custom_chat_%d%02d%02d_%02d%02d%02d.xlsx",
			//	now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
			util.ReturnFile(w, filename, ret)
			auditMsg.WriteString(":")
			auditMsg.WriteString(filename)
		} else {
			auditMsg.WriteString(":")
			auditMsg.WriteString(err.Error())
			util.Return(w, err, nil)
		}
		audit.AddAuditFromRequestAuto(r, auditMsg.String(), retVal)
	}()
	auditMsg.WriteString(localemsg.Get(locale, "ExportCommandsReport"))

	if !validate.IsValidAppID(appid) {
		err = AdminErrors.New(AdminErrors.ErrnoRequestError, "APPID")
		return
	}

	//rId, _ := strconv.Atoi(recordId)

	ret, filename, err = GetCmdImportReport(recordId)
	return
}
func handleExportCmds(w http.ResponseWriter, r *http.Request) {
	//appid := r.URL.Query().Get("appid")
	appid := requestheader.GetAppID(r)
	locale := requestheader.GetLocale(r)
	var err AdminErrors.AdminError
	var ret []byte
	var auditMsg bytes.Buffer

	defer func() {
		retVal := 0
		if err == nil {
			retVal = 1
			now := time.Now()
			filename := fmt.Sprintf("command_%d%02d%02d_%02d%02d%02d.xlsx",
				now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
			util.ReturnFile(w, filename, ret)
			auditMsg.WriteString(":")
			auditMsg.WriteString(filename)
		} else {
			auditMsg.WriteString(":")
			auditMsg.WriteString(err.Error())
			util.Return(w, err, nil)
		}
		audit.AddAuditFromRequestAuto(r, auditMsg.String(), retVal)
	}()
	auditMsg.WriteString(localemsg.Get(locale, "ExportCommands"))

	if !validate.IsValidAppID(appid) {
		err = AdminErrors.New(AdminErrors.ErrnoRequestError, "APPID")
		return
	}

	ret, err = GetFormatCmdByteForExport(appid, locale)
	return
}