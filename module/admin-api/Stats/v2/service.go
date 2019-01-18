package v2

import (
	"emotibot.com/emotigo/module/admin-api/auth"
	"emotibot.com/emotigo/module/admin-api/util/AdminErrors"
	"emotibot.com/emotigo/module/admin-api/util/audit"
	"emotibot.com/emotigo/module/admin-api/util/localemsg"
	"emotibot.com/emotigo/pkg/logger"
)

// GetRobotAuditRecord will get audit record of specific appid
func GetRobotAuditRecord(filter *AuditInput, locale string) (*AuditResult, AdminErrors.AdminError) {
	var userIDPtr *string
	// If query userid is real username, change it to userid
	if filter.UserID != "" {
		userid, err := auth.GetUserID(filter.UserID)
		if err == nil {
			logger.Trace.Printf("Change username to id: %s -> %s\n", filter.UserID, userid)
			userIDPtr = &userid
		} else {
			userIDPtr = &filter.UserID
		}
	}
	modulePtr, opPtr := getModuleOpPtr(filter.Filter)
	logs, count, err := getAuditList(nil, filter.RobotID, userIDPtr, modulePtr, opPtr, filter.Start, filter.End, filter.Page, filter.ListPerPage)
	if err != nil {
		return nil, AdminErrors.New(AdminErrors.ErrnoDBError, err.Error())
	}
	transformLogsWording(logs)

	ret := AuditResult{
		Total:  count,
		Header: robotAuditHeaders[locale],
		Logs:   logs,
	}

	return &ret, nil
}

// GetEnterpriseAuditRecord will get audit record of specific enterprise
func GetEnterpriseAuditRecord(filter *AuditInput, locale string) (*AuditResult, AdminErrors.AdminError) {
	var userIDPtr *string
	// If query userid is real username, change it to userid
	if filter.UserID != "" {
		userid, err := auth.GetUserID(filter.UserID)
		if err == nil {
			logger.Trace.Printf("Change username to id: %s -> %s\n", filter.UserID, userid)
			userIDPtr = &userid
		} else {
			userIDPtr = &filter.UserID
		}
	}
	modulePtr, opPtr := getModuleOpPtr(filter.Filter)
	// only search for empty appid record
	logs, count, err := getAuditList(filter.EnterpriseID, []string{""}, userIDPtr, modulePtr, opPtr, filter.Start, filter.End, filter.Page, filter.ListPerPage)
	if err != nil {
		return nil, AdminErrors.New(AdminErrors.ErrnoDBError, err.Error())
	}
	transformLogsWording(logs)

	ret := AuditResult{
		Total:  count,
		Header: robotAuditHeaders[locale],
		Logs:   logs,
	}

	return &ret, nil
}

// GetSystemAuditRecord will get audit record of specific enterprise
func GetSystemAuditRecord(filter *AuditInput, locale string) (*AuditResult, AdminErrors.AdminError) {
	var userIDPtr *string
	// If query userid is real username, change it to userid
	if filter.UserID != "" {
		userid, err := auth.GetUserID(filter.UserID)
		if err == nil {
			logger.Trace.Printf("Change username to id: %s -> %s\n", filter.UserID, userid)
			userIDPtr = &userid
		} else {
			userIDPtr = &filter.UserID
		}
	}
	modulePtr, opPtr := getModuleOpPtr(filter.Filter)
	logs, count, err := getAuditList([]string{""}, []string{""}, userIDPtr, modulePtr, opPtr, filter.Start, filter.End, filter.Page, filter.ListPerPage)
	if err != nil {
		return nil, AdminErrors.New(AdminErrors.ErrnoDBError, err.Error())
	}
	transformLogsWording(logs)

	ret := AuditResult{
		Total:  count,
		Header: robotAuditHeaders[locale],
		Logs:   logs,
	}

	return &ret, nil
}

func getModuleOpPtr(filter *AuditFilter) ([]string, *string) {
	var modulePtr []string
	var opPtr *string
	if filter != nil {
		if len(filter.Module) > 0 {
			modulePtr = filter.Module
		}
		if filter.Operation != "" {
			opPtr = &filter.Operation
		}
	}
	return modulePtr, opPtr
}

func transformLogsWording(logs []*AuditLog) error {
	userMap := map[string]bool{}
	for idx := range logs {
		userMap[logs[idx].UserID] = true
	}
	users := []string{}
	for key := range userMap {
		users = append(users, key)
	}
	usernameMap, err := auth.GetUserNames(users)
	if err != nil {
		usernameMap = map[string]string{}
	}

	for idx := range logs {
		logs[idx].Module = audit.GetAuditModuleName("", logs[idx].Module)
		logs[idx].Operation = audit.GetAuditOperationName("", logs[idx].Operation)
		if logs[idx].Result > 0 {
			logs[idx].ResultStr = localemsg.Get("", "Success")
		} else {
			logs[idx].ResultStr = localemsg.Get("", "Fail")
		}
		if name, ok := usernameMap[logs[idx].UserID]; ok {
			logs[idx].UserID = name
		}
	}
	return err
}
