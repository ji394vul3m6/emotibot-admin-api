package qi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"emotibot.com/emotigo/module/admin-api/util/AdminErrors"

	"github.com/gorilla/mux"

	"emotibot.com/emotigo/module/admin-api/util"
	"emotibot.com/emotigo/module/admin-api/util/requestheader"
	"emotibot.com/emotigo/module/qic-api/model/v1"
	"emotibot.com/emotigo/module/qic-api/util/general"
	"emotibot.com/emotigo/pkg/logger"
)

// NewGroupReq is the request body schema of the POST or PUT group api
type NewGroupReq struct {
	GroupName       string   `json:"group_name"`
	GroupID         string   `json:"group_id"`
	Description     string   `json:"description"`
	IsEnable        int8     `json:"is_enable"`
	Other           Other    `json:"other"`
	Rules           []string `json:"rules"`
	SilenceRules    []string `json:"silence_rules"`
	SpeedRules      []string `json:"speed_rules"`
	InterposalRules []string `json:"interposal_rules"`
}

//Group transfer NewGroupReq as a model.Group struct, any virtual fields(etc: Other, Rules...) should be handled by the caller.
func (n *NewGroupReq) Group() model.Group {
	return model.Group{
		UUID:            n.GroupID,
		Name:            n.GroupName,
		Description:     n.Description,
		IsEnable:        n.IsEnable != 0,
		Rules:           make([]model.ConversationRule, 0),
		SilenceRules:    make([]model.SilenceRule, 0),
		SpeedRules:      make([]model.SpeedRule, 0),
		InterposalRules: make([]model.InterposalRule, 0),
	}
}

//Other is the condition's json response including custom conditions.
type Other struct {
	Type          int8                     `json:"type"` // it is the ConditionType
	FileName      string                   `json:"file_name"`
	CallTime      int64                    `json:"call_time"`
	Deal          int8                     `json:"deal"`
	Series        string                   `json:"series"`
	StaffID       string                   `json:"staff_id"`
	StaffName     string                   `json:"staff_name"`
	Extension     string                   `json:"extension"`
	Department    string                   `json:"department"`
	CustomerID    string                   `json:"customer_id"`
	CustomerName  string                   `json:"customer_name"`
	CustomerPhone string                   `json:"customer_phone"`
	LeftChannel   string                   `json:"left_channel"`
	RightChannel  string                   `json:"right_channel"`
	CallFrom      int64                    `json:"call_from"`
	CallEnd       int64                    `json:"call_end"`
	CustomColumns map[string][]interface{} `json:"-"`
}

var ReservedConditionKeywords = parseJSONKeys(Other{})

// UnmarshalJSON unmarshal Other with additional custom columns
func (o *Other) UnmarshalJSON(data []byte) error {
	// Check NewCallReq UnmarshalJSON
	type Alias Other
	a := &struct {
		*Alias
	}{
		Alias: (*Alias)(o),
	}
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}

	columns := map[string]interface{}{}
	if err := json.Unmarshal(data, &columns); err != nil {
		return err
	}
	o.CustomColumns = map[string][]interface{}{}
	for col, val := range columns {
		if _, exist := ReservedConditionKeywords[col]; exist {
			continue
		}
		o.CustomColumns[col] = append(o.CustomColumns[col], val)
	}
	return nil
}

// MarshalJSON Other will flatten its CustomColumns map with other fields.
func (o Other) MarshalJSON() ([]byte, error) {
	resp := map[string]interface{}{}
	v := reflect.ValueOf(o)
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		name, opt := getJSONName(tag)
		if name == "-" {
			continue
		}
		if strings.Contains(opt, "omitempty") {
			f := v.Field(i)
			switch f.Kind() {
			case reflect.String:
				if f.String() == "" {
					continue
				}
			case reflect.Float64, reflect.Float32:
				if f.Float() == 0 {
					continue
				}
			case reflect.Int64, reflect.Int32, reflect.Int8:
				if f.Int() == 0 {
					continue
				}
			case reflect.Slice, reflect.Array, reflect.Map:
				if f.IsNil() {
					continue
				}
			}
		}

		resp[name] = v.Field(i).Interface()
	}
	for colName, val := range o.CustomColumns {
		if _, exist := resp[colName]; exist {
			return nil, fmt.Errorf("custom column %s is overlapped with require column", colName)
		}
		resp[colName] = val
	}
	return json.Marshal(resp)
}

var conditionTypDict = map[int8]struct{}{
	model.GroupCondTypOn:  {},
	model.GroupCondTypOff: {},
}

// ValidcondType Return Condition type code(int8) by given input name.
// If none is matched then GroupCondTypOn will return.
func IsValidcondType(typ int8) bool {
	_, exist := conditionTypDict[typ]
	return exist
}

func (o *Other) ToCondition() *model.Condition {
	return &model.Condition{
		Type:          o.Type,
		FileName:      o.FileName,
		Deal:          o.Deal,
		Series:        o.Series,
		StaffID:       o.StaffID,
		StaffName:     o.StaffName,
		Extension:     o.Extension,
		Department:    o.Department,
		CustomerID:    o.CustomerID,
		CustomerName:  o.CustomerName,
		CustomerPhone: o.CustomerPhone,
		LeftChannel:   int8(RoleMatcherTyp(o.LeftChannel)),
		RightChannel:  int8(RoleMatcherTyp(o.RightChannel)),
		CallStart:     o.CallFrom,
		CallEnd:       o.CallEnd,
	}
}

func toOther(cond *model.Condition, customCond map[string][]interface{}) Other {
	return Other{
		Type:          cond.Type,
		FileName:      cond.FileName,
		Deal:          cond.Deal,
		Series:        cond.Series,
		StaffID:       cond.StaffID,
		StaffName:     cond.StaffName,
		Extension:     cond.Extension,
		Department:    cond.Department,
		CustomerID:    cond.CustomerID,
		CustomerName:  cond.CustomerName,
		CustomerPhone: cond.CustomerPhone,
		LeftChannel:   RoleMatcherString(int(cond.LeftChannel)),
		RightChannel:  RoleMatcherString(int(cond.RightChannel)),
		CallFrom:      cond.CallStart,
		CallEnd:       cond.CallEnd,
		CustomColumns: customCond,
	}
}

// func groupInReqToGroupWCond(inreq *GroupInReq) *model.GroupWCond {
// 	group := &model.GroupWCond{
// 		UUID:            inreq.UUID,
// 		Name:            inreq.Name,
// 		Enabled:         inreq.Enabled,
// 		Speed:           inreq.Speed,
// 		SlienceDuration: inreq.SlienceDuration,
// 		Condition:       inreq.Condition.ToGroupCondition(),
// 		CreateTime:      inreq.CreateTime,
// 		Description:     inreq.Description,
// 		RuleCount:       inreq.RuleCount,
// 	}

// 	simpleRules := []model.SimpleConversationRule{}
// 	for _, ruleID := range inreq.Rules {
// 		simpleRule := model.SimpleConversationRule{
// 			UUID: ruleID,
// 		}
// 		simpleRules = append(simpleRules, simpleRule)
// 	}
// 	group.Rules = &simpleRules
// 	return group
// }

// handleCreateGroup handle the group creation controller logic.
//
// **FIXME: It ignore the other rules and only creates group with conversation rules,**
// but it should be a minor issue since UI separate the creation of group and its rules.
func handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var (
		reqBody NewGroupReq
	)
	err := util.ReadJSON(r, &reqBody)
	if err != nil {
		util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("Bad Request Body, %v", err))
		return
	}
	if !IsValidcondType(reqBody.Other.Type) {
		util.ReturnError(w, AdminErrors.ErrnoRequestError, "require Other Condition")
		return
	}

	group := reqBody.Group()
	group.EnterpriseID = requestheader.GetEnterpriseID(r)
	if reqBody.Rules != nil {
		var ruleTotal int64
		ruleTotal, group.Rules, err = getConversationRulesBy(&model.ConversationRuleFilter{
			UUID: reqBody.Rules,
		})
		if err != nil {
			util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("get rules failed, %v", err))
			return
		}
		if int(ruleTotal) != len(reqBody.Rules) {
			util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("request rules %v, but system only have %d", reqBody.Rules, ruleTotal))
			return
		}
	}
	if len(reqBody.SilenceRules) > 0 {

	}
	if len(reqBody.SpeedRules) > 0 {

	}
	if len(reqBody.InterposalRules) > 0 {

	}

	condition := reqBody.Other.ToCondition()
	customConditions := reqBody.Other.CustomColumns
	group, err = NewGroupWithAllConditions(group, *condition, customConditions)
	if err != nil {
		util.ReturnError(w, AdminErrors.ErrnoDBError, fmt.Sprintf("new group with conditions failed, %v", err))
		return
	}
	response := struct {
		UUID string `json:"group_id"`
	}{
		UUID: group.UUID,
	}

	util.WriteJSON(w, response)
}

func handleGetGroups(w http.ResponseWriter, r *http.Request) {
	type GroupsResponse struct {
		Paging general.Paging `json:"paging"`
		Data   []GroupResp    `json:"data"`
	}
	values := r.URL.Query()
	filter, err := parseGroupFilter(&values)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	total, groups, err := GroupResps(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := GroupsResponse{
		Paging: general.Paging{
			Page:  filter.Page,
			Limit: filter.Limit,
			Total: total,
		},
		Data: groups,
	}

	util.WriteJSON(w, response)
}

func parseID(r *http.Request) (id string) {
	vars := mux.Vars(r)
	return vars["id"]
}

var methodDict = map[int8]string{
	model.RuleMethodPositive: "positive",
	model.RuleMethodNegative: "negative",
}

func handleGetGroup(w http.ResponseWriter, r *http.Request, group *model.Group) {
	type GeneralRuleResp struct {
		UUID string `json:"rule_id"`
		Name string `json:"rule_name"`
	}
	type RuleResp struct {
		GeneralRuleResp
		Method string `json:"method"`
	}
	type GroupDetailResp struct {
		GroupResp
		Rules        []RuleResp        `json:"rules"`
		SilenceRules []GeneralRuleResp `json:"silence_rules"`
		SpeedRules   []GeneralRuleResp `json:"speed_rules"`
		Interposal   []GeneralRuleResp `json:"interposal_rules"`
	}
	var err error
	group, err = GetGroupRules(*group)
	customData, err := customConditionsOfGroup(group.ID)
	if err != nil {
		util.ReturnError(w, AdminErrors.ErrnoDBError, fmt.Sprintf("Get conditions of group failed, %v", err))
		return
	}
	cond, err := getConditionOfGroup(group.ID)
	if err != nil {
		util.ReturnError(w, AdminErrors.ErrnoDBError, fmt.Sprintf("get con"))
		return
	}
	ruleCount := len(group.Rules) + len(group.SpeedRules) + len(group.SilenceRules) + len(group.InterposalRules)
	var resp = GroupDetailResp{
		GroupResp: GroupResp{
			GroupID:     group.UUID,
			GroupName:   group.Name,
			Other:       toOther(cond, customData),
			CreateTime:  group.CreatedTime,
			Description: group.Description,
			RuleCount:   ruleCount,
		},
		Rules:        make([]RuleResp, 0, ruleCount),
		SilenceRules: make([]GeneralRuleResp, 0),
		SpeedRules:   make([]GeneralRuleResp, 0),
		Interposal:   make([]GeneralRuleResp, 0),
	}
	if group.IsEnable {
		resp.IsEnable = 1
	}

	for _, r := range group.Rules {
		resp.Rules = append(resp.Rules, RuleResp{
			GeneralRuleResp: GeneralRuleResp{
				UUID: r.UUID,
				Name: r.Name,
			},
			Method: methodDict[r.Method],
		})
	}
	for _, r := range group.SilenceRules {
		resp.SilenceRules = append(resp.SilenceRules, GeneralRuleResp{
			UUID: r.UUID,
			Name: r.Name,
		})
	}
	for _, r := range group.SpeedRules {
		resp.SpeedRules = append(resp.SpeedRules, GeneralRuleResp{
			UUID: r.UUID,
			Name: r.Name,
		})
	}
	for _, r := range group.InterposalRules {
		resp.Interposal = append(resp.Interposal, GeneralRuleResp{
			UUID: r.UUID,
			Name: r.Name,
		})
	}
	util.WriteJSON(w, resp)

}

func handleUpdateGroup(w http.ResponseWriter, r *http.Request, group *model.GroupWCond) {
	var reqBody NewGroupReq
	err := util.ReadJSON(r, &reqBody)
	if err != nil {
		util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("Bad Request Body, %v", err))
		return
	}
	if !IsValidcondType(reqBody.Other.Type) {
		util.ReturnError(w, AdminErrors.ErrnoRequestError, "require Other Condition")
		return
	}

	newGroup := reqBody.Group()
	newGroup.EnterpriseID = group.Enterprise
	newGroup.UUID = group.UUID
	newGroup.Condition = reqBody.Other.ToCondition()
	customConditions := reqBody.Other.CustomColumns
	notDeleted := 0
	if len(reqBody.Rules) > 0 {
		var (
			total int64
			rules []model.ConversationRule
		)

		total, rules, err = getConversationRulesBy(&model.ConversationRuleFilter{
			Enterprise: group.Enterprise,
			Severity:   -1,
			UUID:       reqBody.Rules,
			IsDeleted:  0,
		})
		if err != nil {
			util.ReturnError(w, AdminErrors.ErrnoDBError, fmt.Sprintf("get rules failed, %v", err))
		}
		if int(total) != len(reqBody.Rules) {
			util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("some rules does not exist."))
			return
		}

		ruleIDs := make([]uint64, 0, total)
		for _, v := range rules {
			ruleIDs = append(ruleIDs, uint64(v.ID))
		}

		levValid, err := CheckIntegrity(LevRule, ruleIDs)
		if err != nil {
			util.ReturnError(w, AdminErrors.ErrnoDBError, fmt.Sprintf("call check integrity failed. %v", err))
			return
		}
		for idx, lev := range levValid {
			if !lev.Valid {
				invalidRule := rules[idx]
				util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("%s(%s) is not complete", invalidRule.Name, invalidRule.UUID))
				return
			}
		}

		newGroup.Rules = rules
	}
	if len(reqBody.SilenceRules) > 0 {
		rules, err := GetRuleSilences(&model.GeneralQuery{
			UUID:       reqBody.SilenceRules,
			Enterprise: &newGroup.EnterpriseID,
			IsDelete:   &notDeleted,
		}, nil)
		if err != nil {
			util.ReturnError(w, AdminErrors.ErrnoDBError, fmt.Sprintf("get silence rules failed, %v", err))
			return
		}
		if len(reqBody.SilenceRules) != len(rules) {
			util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("invalid silence rule input"))
			return
		}
		for _, rule := range rules {
			newGroup.SilenceRules = append(newGroup.SilenceRules, *rule)
		}
	}
	if len(reqBody.SpeedRules) > 0 {
		rules, err := GetRuleSpeeds(&model.GeneralQuery{
			UUID:       reqBody.SpeedRules,
			Enterprise: &newGroup.EnterpriseID,
			IsDelete:   &notDeleted,
		}, nil)
		if err != nil {
			util.ReturnError(w, AdminErrors.ErrnoDBError, fmt.Sprintf("get speed rules failed, %v", err))
			return
		}
		if len(reqBody.SpeedRules) != len(rules) {
			util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("invalid speed rules input"))
			return
		}
		for _, rule := range rules {
			newGroup.SpeedRules = append(newGroup.SpeedRules, *rule)
		}
	}
	if len(reqBody.InterposalRules) > 0 {
		rules, err := GetRuleInterposals(&model.GeneralQuery{
			UUID:       reqBody.InterposalRules,
			Enterprise: &newGroup.EnterpriseID,
			IsDelete:   &notDeleted,
		}, nil)
		if err != nil {
			util.ReturnError(w, AdminErrors.ErrnoDBError, fmt.Sprintf("get interposal rules failed, %v", err))
			return
		}
		if len(reqBody.InterposalRules) != len(rules) {
			util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("invalid interposal rules input"))
			return
		}
		for _, rule := range rules {
			newGroup.InterposalRules = append(newGroup.InterposalRules, *rule)
		}
	}

	err = UpdateGroup(newGroup, customConditions)
	if err != nil {
		util.ReturnError(w, AdminErrors.ErrnoDBError, fmt.Sprintf("update group failed, %v", err))
		return
	}
}

// handleUpdateGroupBaisc is the
func handleUpdateGroupBasic(w http.ResponseWriter, r *http.Request, group *model.Group) {
	err := UpdateGroupBasic(group)
	if err != nil {
		util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("update group basic failed, %v", err))
		return
	}
}

func handleDeleteGroup(w http.ResponseWriter, r *http.Request, group *model.GroupWCond) {
	err := DeleteGroup(group.UUID)
	if err != nil {
		util.ReturnError(w, AdminErrors.ErrnoDBError, fmt.Sprintf("delete group failed, %v", err))
		return
	}
}

func handleGetGroupsByFilter(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	filter, err := parseGroupFilter(&values)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	total, groups, err := GroupResps(filter)
	if err != nil {
		logger.Error.Printf("error while get groups by filter in handleGetGroupsByFilter, reason: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type GroupsResponse struct {
		Paging general.Paging `json:"paging"`
		Data   []GroupResp    `json:"data"`
	}
	response := GroupsResponse{
		Paging: general.Paging{
			Page:  filter.Page,
			Total: total,
			Limit: filter.Limit,
		},
		Data: groups,
	}

	util.WriteJSON(w, response)
}

func parseGroupFilter(values *url.Values) (filter *model.GroupFilter, err error) {
	filter = &model.GroupFilter{}
	filter.FileName = values.Get("file_name")
	filter.Series = values.Get("series")
	filter.StaffID = values.Get("staff_id")
	filter.StaffName = values.Get("staff_name")
	filter.Extension = values.Get("extension")
	filter.Department = values.Get("department")
	filter.CustomerID = values.Get("customer_id")
	filter.CustomerName = values.Get("customer_name")
	filter.CustomerPhone = values.Get("customer_phone")

	deleted := int8(0)
	filter.Delete = &deleted

	dealStr := values.Get("deal")
	if dealStr != "" {
		deal, ierr := strconv.Atoi(dealStr)
		filter.Deal = &deal
		if err != nil {
			return filter, ierr
		}
	}

	callStartStr := values.Get("call_start")
	if callStartStr != "" {
		filter.CallStart, err = strconv.ParseInt(callStartStr, 10, 64)
		if err != nil {
			return
		}
	}

	callEndStr := values.Get("call_end")
	if callEndStr != "" {
		filter.CallEnd, err = strconv.ParseInt(callEndStr, 10, 64)
		if err != nil {
			return
		}
	}

	pageStr := values.Get("page")
	if pageStr != "" {
		filter.Page, err = strconv.Atoi(pageStr)
		if err != nil {
			return
		}
	}

	limitStr := values.Get("limit")
	if limitStr != "" {
		filter.Limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return
		}
	} else {
		filter.Limit = 0
	}
	return
}
func simpleGroupRequest(next func(w http.ResponseWriter, r *http.Request, group *model.Group)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const GroupIDKey = "group_id"
		groupUUID := mux.Vars(r)[GroupIDKey]
		if GroupIDKey == "" {
			util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("require %s in path", GroupIDKey))
			return
		}
		enterprise := requestheader.GetEnterpriseID(r)
		if enterprise == "" {
			util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("empty enterprise ID"))
			return
		}
		groups, err := groups(nil, model.GroupQuery{
			UUID:         []string{groupUUID},
			EnterpriseID: enterprise,
		})
		if len(groups) == 0 {
			util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("id '%s' is not exist", groupUUID))
			return
		}
		g := groups[0]
		if err != nil {
			util.ReturnError(w, AdminErrors.ErrnoDBError, fmt.Sprintf("get group failed, %v", err))
			return
		}
		next(w, r, &g)
	}
}
func groupRequest(next func(w http.ResponseWriter, r *http.Request, group *model.GroupWCond)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const GroupIDKey = "group_id"
		groupUUID := mux.Vars(r)[GroupIDKey]
		if GroupIDKey == "" {
			util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("require %s in path", GroupIDKey))
			return
		}
		enterprise := requestheader.GetEnterpriseID(r)
		if enterprise == "" {
			util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("empty enterprise ID"))
			return
		}
		g, err := GetGroupBy(groupUUID)
		if err == ErrNotFound {
			util.ReturnError(w, AdminErrors.ErrnoRequestError, fmt.Sprintf("id '%s' is not exist", groupUUID))
			return
		}
		if err != nil {
			util.ReturnError(w, AdminErrors.ErrnoDBError, fmt.Sprintf("get group failed, %v", err))
			return
		}
		g.UUID = groupUUID

		next(w, r, g)
	}
}
