package FAQ

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"emotibot.com/emotigo/module/admin-api/ApiError"
	"emotibot.com/emotigo/module/admin-api/util"
	"emotibot.com/emotigo/module/admin-api/util/audit"
	"emotibot.com/emotigo/module/admin-api/util/requestheader"
	"emotibot.com/emotigo/pkg/logger"
)

var (
	// ModuleInfo is needed for module define
	ModuleInfo util.ModuleInfo
)

func init() {
	ModuleInfo = util.ModuleInfo{
		ModuleName: "faq",
		EntryPoints: []util.EntryPoint{
			util.NewEntryPoint("GET", "question/{qid}/similar-questions", []string{"edit"}, handleQuerySimilarQuestions),
			util.NewEntryPoint("POST", "question/{qid}/similar-questions", []string{"edit"}, handleUpdateSimilarQuestions),
			util.NewEntryPoint("DELETE", "question/{qid}/similar-questions", []string{"edit"}, handleDeleteSimilarQuestions),
			util.NewEntryPoint("GET", "questions/search", []string{"view"}, handleSearchQuestion),
			util.NewEntryPoint("GET", "questions/filter", []string{"view"}, handleQuestionFilter),

			util.NewEntryPoint("GET", "RFQuestions", []string{"view"}, handleGetRFQuestions),
			util.NewEntryPoint("POST", "RFQuestions", []string{"edit"}, handleSetRFQuestions),

			util.NewEntryPoint("GET", "category/{cid}/questions", []string{"view"}, handleCategoryQuestions),
			util.NewEntryPoint("GET", "categories", []string{"view"}, handleGetCategories),
			util.NewEntryPoint("POST", "category/{id}", []string{"edit"}, handleUpdateCategories),
			util.NewEntryPoint("PUT", "category", []string{"edit"}, handleAddCategory),
			util.NewEntryPoint("DELETE", "category/{id}", []string{"edit"}, handleDeleteCategory),

			util.NewEntryPoint("GET", "labels", []string{"view"}, handleGetLabels),
			util.NewEntryPoint("PUT", "label/{id}", []string{"view"}, handleUpdateLabel),
			util.NewEntryPoint("POST", "label", []string{"view"}, handleAddLabel),
			util.NewEntryPoint("DELETE", "label/{id}", []string{"view"}, handleDeleteLabel),
			util.NewEntryPoint("PUT", "question/{qid}/answer/{aid}/label", []string{"edit"}, handleUpdateQuestionLabel),

			util.NewEntryPoint("GET", "rules", []string{"view"}, handleGetRules),
			util.NewEntryPoint("GET", "rule/{id}", []string{"edit"}, handleGetRule),
			util.NewEntryPoint("PUT", "rule/{id}", []string{"edit"}, handleUpdateRule),
			util.NewEntryPoint("POST", "rule", []string{"create"}, handleAddRule),
			util.NewEntryPoint("DELETE", "rule/{id}", []string{"view"}, handleDeleteRule),

			util.NewEntryPoint("GET", "label/{id}/rules", []string{"view"}, handleGetRulesOfLabel),
			util.NewEntryPoint("GET", "rule/{id}/labels", []string{"view"}, handleGetLabelsOfRule),

			// util.NewEntryPoint("POST", "rule/{id}/label/add", []string{"edit"}, handleAddLabelToRule),
			// util.NewEntryPoint("DELETE", "rule/{id}/label/{id}", []string{"edit"}, handleDeleteLabelFromRule),
			// util.NewEntryPoint("POST", "label/{id}/rule/add", []string{"edit"}, handleAddRuleToLabel),
			// util.NewEntryPoint("DELETE", "label/{id}/rule/{id}", []string{"edit"}, handleDeleteRuleFromLabel),

			util.NewEntryPoint("GET", "tag-types", []string{"view"}, handleGetTagTypes),
			util.NewEntryPoint("GET", "tag-type/{id}", []string{"view"}, handleGetTagType),
			util.NewEntryPointWithVer("GET", "tag-types", []string{"view"}, handleGetTagTypesV2, 2),
			util.NewEntryPointWithVer("GET", "tag-type/{id}", []string{"view"}, handleGetTagTypeV2, 2),
		},
	}
}

func handleAddCategory(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)
	userID := requestheader.GetUserID(r)
	userIP := requestheader.GetUserIP(r)

	name := r.FormValue("categoryname")
	parentID, err := strconv.Atoi(r.FormValue("parentid"))
	if err != nil || name == "" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	parentCategory, err := GetAPICategory(appid, parentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var newCatetory *APICategory
	var path string
	if parentCategory == nil {
		newCatetory, err = AddAPICategory(appid, name, 0, 1)
		path = name
	} else {
		newCatetory, err = AddAPICategory(appid, name, parentID, parentCategory.Level+1)
		paths := strings.Split(parentCategory.Path, "/")
		path = strings.Join(append(paths[1:], name), "/")
	}
	auditMessage := fmt.Sprintf("[%s]:%s", util.Msg["Category"], path)
	auditRet := 1
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		auditRet = 0
	} else {
		util.WriteJSON(w, newCatetory)
	}
	enterpriseID := requestheader.GetEnterpriseID(r)
	audit.AddAuditLog(enterpriseID, appid, userID, userIP, audit.AuditModuleFAQ, audit.AuditOperationEdit, auditMessage, auditRet)
}

func handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)
	userID := requestheader.GetUserID(r)
	userIP := requestheader.GetUserIP(r)
	categoryID, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	origCategory, err := GetAPICategory(appid, categoryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if origCategory == nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	paths := strings.Split(origCategory.Path, "/")
	path := strings.Join(paths[1:], "/")

	count, err := GetCategoryQuestionCount(appid, origCategory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = DeleteAPICategory(appid, origCategory)
	fmtStr := "[%s]:%s，" + util.Msg["DeleteCategoryDesc"]
	auditMessage := fmt.Sprintf(fmtStr, util.Msg["Category"], path, count)
	auditRet := 1
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		auditRet = 0
	}
	enterpriseID := requestheader.GetEnterpriseID(r)
	audit.AddAuditLog(enterpriseID, appid, userID, userIP, audit.AuditModuleFAQ, audit.AuditOperationEdit, auditMessage, auditRet)
	util.ConsulUpdateFAQ(appid)
}

func handleUpdateCategories(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)
	userID := requestheader.GetUserID(r)
	userIP := requestheader.GetUserIP(r)
	categoryID, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	newName := r.FormValue("categoryname")
	if newName == "" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	origCategory, err := GetAPICategory(appid, categoryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if origCategory == nil {
		http.Error(w, "", http.StatusBadRequest)
	}
	origCategory.Name = newName
	err = UpdateAPICategoryName(appid, categoryID, newName)

	origPaths := strings.Split(origCategory.Path, "/")
	origPath := strings.Join(origPaths[1:], "/")
	newPath := strings.Join(append(origPaths[1:len(origPaths)-1], newName), "/")

	auditMessage := fmt.Sprintf("[%s]:%s=>%s", util.Msg["Category"], origPath, newPath)
	auditRet := 1
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		auditRet = 0
	}
	enterpriseID := requestheader.GetEnterpriseID(r)
	audit.AddAuditLog(enterpriseID, appid, userID, userIP, audit.AuditModuleFAQ, audit.AuditOperationEdit, auditMessage, auditRet)
}

func handleGetCategories(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)
	categories, err := GetAPICategories(appid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	util.WriteJSON(w, categories)
}

func handleQuerySimilarQuestions(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func handleUpdateSimilarQuestions(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)
	qid, err := util.GetMuxIntVar(r, "qid")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	proccessStatus := 0
	userID := requestheader.GetUserID(r)
	userIP := requestheader.GetUserIP(r)

	questions, err := selectQuestions([]int{qid}, appid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if len(questions) == 0 {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	var question = questions[0]
	questionCategory, err := GetCategory(question.CategoryID, appid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	categoryName, err := questionCategory.FullName(appid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	auditMessage := fmt.Sprintf("[相似问题]:[%s][%s]:", categoryName, question.Content)
	// select origin Similarity Questions for audit log
	originSimilarityQuestions, err := selectSimilarQuestions(qid, appid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	body := SimilarQuestionReqBody{}
	if err = util.ReadJSON(r, &body); err != nil {
		logger.Info.Printf("Bad request when loading from input: %s", err.Error())
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	sqs := body.SimilarQuestions

	// update similar questions
	err = updateSimilarQuestions(qid, appid, userID, sqs)
	if err != nil {
		enterpriseID := requestheader.GetEnterpriseID(r)
		audit.AddAuditLog(enterpriseID, appid, userID, userIP, audit.AuditModuleFAQ, audit.AuditOperationEdit, "更新相似问失败", proccessStatus)
		logger.Error.Println(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	//sqsStr 移除了沒更動的相似問
	var sqsStr []string
	//contentMatching 邏輯: 移除掉一模一樣的新舊相似問內容, 來寫audit log
contentMatching:
	for i := 0; i < len(sqs); i++ {
		sq := sqs[i].Content
		for j := len(originSimilarityQuestions) - 1; j >= 0; j-- {
			oldSq := originSimilarityQuestions[j]
			if sq == oldSq {
				originSimilarityQuestions = append(originSimilarityQuestions[:j], originSimilarityQuestions[j+1:]...)
				continue contentMatching
			}
		}
		sqsStr = append(sqsStr, sq)
	}
	sort.Strings(originSimilarityQuestions)
	sort.Strings(sqsStr)

	proccessStatus = 1
	operation := audit.AuditOperationEdit
	//當全部都是新的(原始的被扣完)行為要改成新增, 全部都是舊的(新的是空的)行為要改成刪除
	if len(originSimilarityQuestions) == 0 {
		operation = audit.AuditOperationAdd
		auditMessage += fmt.Sprintf("%s", strings.Join(sqsStr, ";"))
	} else if len(sqsStr) == 0 {
		operation = audit.AuditOperationDelete
		auditMessage += fmt.Sprintf("%s", strings.Join(originSimilarityQuestions, ";"))
	} else {
		auditMessage += fmt.Sprintf("%s=>%s", strings.Join(originSimilarityQuestions, ";"), strings.Join(sqsStr, ";"))

	}
	enterpriseID := requestheader.GetEnterpriseID(r)
	audit.AddAuditLog(enterpriseID, appid, userID, userIP, audit.AuditModuleFAQ, operation, auditMessage, proccessStatus)

}

func handleDeleteSimilarQuestions(w http.ResponseWriter, r *http.Request) {
	// TODO
}

// search question by exactly matching content
func handleSearchQuestion(w http.ResponseWriter, r *http.Request) {
	content := r.FormValue("content")
	appid := requestheader.GetAppID(r)
	question, err := searchQuestionByContent(content, appid)
	if err == util.ErrSQLRowNotFound {
		http.Error(w, "", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error.Printf("searching Question by content [%s] failed, %s", content, err)
		return
	}
	util.WriteJSON(w, question)
}

//Retrun JSON Formatted RFQuestion array, if question is invalid, id & categoryId will be 0
func handleGetRFQuestions(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)
	questions, err := GetRFQuestions(appid)
	if err != nil {
		logger.Error.Printf("Get RFQuestions failed, %v\n", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	util.WriteJSON(w, questions)
}

func handleSetRFQuestions(w http.ResponseWriter, r *http.Request) {
	var args UpdateRFQuestionsArgs
	appid := requestheader.GetAppID(r)
	err := util.ReadJSON(r, &args)
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	if err = SetRFQuestions(args.Contents, appid); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error.Println(err)
		return
	}

}

func handleCategoryQuestions(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)
	id, err := util.GetMuxIntVar(r, "cid")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	category, err := GetCategory(id, appid)
	if err == sql.ErrNoRows {
		http.Error(w, "", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error.Println(err)
		return
	}
	includeSub := r.URL.Query().Get("includeSubCat")
	var categories []Category
	if includeSub == "true" {
		categories, err = category.SubCats(appid)
		if err != nil {
			http.Error(w, "", http.StatusInternalServerError)
			logger.Error.Println(err)
			return
		}

	}
	//Add category itself into total
	categories = append(categories, category)
	questions, err := GetQuestionsByCategories(categories, appid)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		logger.Error.Println(err)
		return
	}

	util.WriteJSON(w, questions)
}

func handleQuestionFilter(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)
	// parse QueryCondition
	condition, err := ParseCondition(r)
	if err != nil {
		logger.Error.Printf("Error happened while parsing query options %s", err.Error())
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	// fetch question ids and total row number
	qids, aids, err := DoFilter(condition, appid)

	if err != nil {
		logger.Error.Printf("Error happened while Filter questions %s", err.Error())
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// paging qids
	start := condition.CurPage * condition.Limit
	end := start + condition.Limit

	// fetch returned question and answers
	type Response struct {
		CurPage     string     `json:"CurPage"`
		Questions   []Question `json:"QueryResult"`
		PageNum     float64    `json:"TotalNum"`
		QuestionNum int        `json:"TotalQuestionNum"`
	}

	var pagedQIDs []int
	var pagedAIDs [][]string
	if len(qids) == 0 {
		response := Response{
			CurPage:     "0",
			Questions:   make([]Question, 0),
			PageNum:     0,
			QuestionNum: 0,
		}

		util.WriteJSON(w, response)
		return
	} else if len(qids) < condition.Limit {
		pagedQIDs = qids
		pagedAIDs = aids
	} else if len(qids) < end {
		end = len(qids)
		pagedQIDs = qids[start:end]
		pagedAIDs = aids[start:end]
	} else {
		pagedQIDs = qids[start:end]
		pagedAIDs = aids[start:end]
	}

	questions, err := DoFetch(pagedQIDs, pagedAIDs, appid)
	if err != nil {
		logger.Error.Printf("Error happened Fetch questions %s", err.Error())
	}

	total := len(qids)
	pageNum := math.Floor(float64(total / condition.Limit))

	response := Response{
		CurPage:     strconv.Itoa(condition.CurPage),
		Questions:   questions,
		PageNum:     pageNum,
		QuestionNum: total,
	}

	util.WriteJSON(w, response)
}

func handleGetTagTypes(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)

	tag, err := GetTagTypes(appid, 1)
	if err != nil {
		util.WriteJSONWithStatus(w, util.GenRetObj(ApiError.DB_ERROR, err.Error()), http.StatusInternalServerError)
	} else {
		util.WriteJSON(w, util.GenRetObj(ApiError.SUCCESS, tag))
	}
}
func handleGetTagType(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)
	id, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		util.WriteJSONWithStatus(w, util.GenRetObj(ApiError.DB_ERROR, err.Error()), http.StatusBadRequest)
		return
	}

	tag, err := GetTagType(appid, id, 1)
	if err != nil {
		util.WriteJSONWithStatus(w, util.GenRetObj(ApiError.DB_ERROR, err.Error()), http.StatusInternalServerError)
	} else {
		util.WriteJSON(w, util.GenRetObj(ApiError.SUCCESS, tag))
	}
}

func handleGetTagTypesV2(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)

	tag, err := GetTagTypes(appid, 2)
	if err != nil {
		util.WriteJSONWithStatus(w, util.GenRetObj(ApiError.DB_ERROR, err.Error()), http.StatusInternalServerError)
	} else {
		util.WriteJSON(w, util.GenRetObj(ApiError.SUCCESS, tag))
	}
}
func handleGetTagTypeV2(w http.ResponseWriter, r *http.Request) {
	appid := requestheader.GetAppID(r)
	id, err := util.GetMuxIntVar(r, "id")
	if err != nil {
		util.WriteJSONWithStatus(w, util.GenRetObj(ApiError.DB_ERROR, err.Error()), http.StatusBadRequest)
		return
	}

	tag, err := GetTagType(appid, id, 2)
	if err != nil {
		util.WriteJSONWithStatus(w, util.GenRetObj(ApiError.DB_ERROR, err.Error()), http.StatusInternalServerError)
	} else {
		util.WriteJSON(w, util.GenRetObj(ApiError.SUCCESS, tag))
	}
}

func handleUpdateQuestionLabel(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	errno := ApiError.SUCCESS
	var err error
	var ret interface{}
	defer func() {
		if err != nil {
			ret = err.Error()
		}
		util.WriteJSONWithStatus(w, util.GenRetObj(errno, ret), status)

		// TODO: add audit log here
	}()

	appid := requestheader.GetAppID(r)
	questionID, err := util.GetMuxIntVar(r, "qid")
	if err != nil {
		status, errno, ret = http.StatusBadRequest, ApiError.REQUEST_ERROR, "invalid qid"
		return
	}
	answerID, err := util.GetMuxIntVar(r, "aid")
	if err != nil {
		status, errno, ret = http.StatusBadRequest, ApiError.REQUEST_ERROR, "invalid aid"
		return
	}
	labelStr := strings.TrimSpace(r.FormValue("labels"))
	var labelIDs []int
	if labelStr != "" {
		labelStrSlice := strings.Split(labelStr, ",")
		labelIDs = make([]int, len(labelStrSlice))
		for idx := range labelStrSlice {
			labelIDs[idx], err = strconv.Atoi(labelStrSlice[idx])
			if err != nil {
				status, errno, ret = http.StatusBadRequest, ApiError.REQUEST_ERROR, "invalid label ids"
				return
			}
		}
	} else {
		labelIDs = []int{}
	}
	logger.Trace.Printf("Update label of answer [%d] to [%s]", answerID, labelStr)

	errno, err = UpdateQALabel(appid, questionID, answerID, labelIDs)
	if err != nil {
		status = http.StatusInternalServerError
	}
	return
}
