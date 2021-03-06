package FAQ

import (
	"fmt"
	"net/http"
	"strconv"

	"emotibot.com/emotigo/module/vipshop-admin/util"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

//Retrun JSON Formatted RFQuestion array, if question is invalid, id & categoryId will be 0
func handleGetRFQuestions(ctx iris.Context) {
	appid := util.GetAppID(ctx)
	questions, err := GetRFQuestions(appid)
	if err != nil {
		util.LogError.Printf("Get RFQuestions failed, %v\n", err)
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}
	ctx.JSON(questions)
}

func handleSetRFQuestions(ctx iris.Context) {
	var args UpdateRFQuestionsArgs
	appid := util.GetAppID(ctx)
	err := ctx.ReadJSON(&args)
	if err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}

	// select old RFQuestions for audit log
	oldRFQuestions, err := GetRFQuestions(appid)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}
	oldRFQuestionsContent := make([]string, len(oldRFQuestions))
	for i := range oldRFQuestions {
		oldRFQuestionsContent[i] = oldRFQuestions[i].Content
	}

	auditRet := 1
	if err = SetRFQuestions(args.Contents, appid); err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		util.LogError.Println(err)
		auditRet = 0
	}

	// write audit log
	writeAuditLog(ctx, auditRet, oldRFQuestionsContent, args.Contents)
}

func writeAuditLog(ctx context.Context, result int, oldRFQuestios, newRFQuestions []string) (err error) {
	// prepare map
	var oldRFQuestiosMap map[string]bool = make(map[string]bool)
	var newRFQuestionMap map[string]bool = make(map[string]bool)

	for i := range oldRFQuestios {
		oldRFQuestiosMap[oldRFQuestios[i]] = true
	}

	for i := range newRFQuestions {
		newRFQuestionMap[newRFQuestions[i]] = true
	}

	deletedRFQuestionLog := prepareAuditLog(oldRFQuestiosMap, newRFQuestionMap)
	addedRFQuestionLog := prepareAuditLog(newRFQuestionMap, oldRFQuestiosMap)

	userID := util.GetUserID(ctx)
	userIP := util.GetUserIP(ctx)

	if deletedRFQuestionLog != "" {
		err = util.AddAuditLog(userID, userIP, util.AuditModuleRFQuestion, util.AuditOperationDelete, deletedRFQuestionLog, result)
		if err != nil {
			return
		}
	}

	if addedRFQuestionLog != "" {
		err = util.AddAuditLog(userID, userIP, util.AuditModuleRFQuestion, util.AuditOperationAdd, addedRFQuestionLog, result)
		if err != nil {
			return
		}
	}
	return
}

// diff two map and append the content to create audit log
func prepareAuditLog(sourceMap, targetMap map[string]bool) (log string) {
	var first bool = false
	for key, _ := range sourceMap {
		_, ok := targetMap[key]
		if !ok {
			if !first {
				log = "[解决/未解决问题]："
				log = fmt.Sprintf("%s%s", log, key)
				first = true
			} else {
				log = fmt.Sprintf("%s;%s", log, key)
			}
		}
	}
	return
}

func handleCategoryRFQuestions(ctx context.Context) {
	cid := ctx.Params().Get("cid")
	appid := util.GetAppID(ctx)
	categoryID, err := strconv.ParseInt(cid, 10, 64)
	if err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		fmt.Fprintln(ctx, "category id invalid, "+err.Error())
		return
	}
	catTree, err := NewCategoryTree(appid)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		fmt.Fprintln(ctx, "categories tree build failed, "+err.Error())
		return
	}
	categories := catTree.SubCategories(categoryID)
	var categoryIDGroup []int64
	for _, c := range categories {
		categoryIDGroup = append(categoryIDGroup, int64(c.ID))
	}
	questionsDict, err := GetRFQuestionsByCategoryId(appid, categoryIDGroup)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		fmt.Fprintln(ctx, "err "+err.Error())
		return
	}
	output := []struct {
		Content    string `json:"content"`
		CategoryId int64  `json:"CategoryId"`
	}{}
	for cat, questions := range questionsDict {
		for _, q := range questions {
			vaildRFQ := struct {
				Content    string `json:"content"`
				CategoryId int64  `json:"CategoryId"`
			}{}
			vaildRFQ.Content = q
			vaildRFQ.CategoryId = cat
			output = append(output, vaildRFQ)
		}
	}

	ctx.JSON(output)
}

//1. check input, return 400 if  size <0 || size > 30
//2. Filter RFQ & stdQ base on input
func handleRFQValidation(ctx context.Context) {
	type parameters struct {
		RFQuestions []string `json:"RFQuestions"`
	}
	var input parameters
	err := ctx.ReadJSON(&input)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		fmt.Fprintf(ctx, "JSON unmarshal error, %vs", err)
		return
	}
	if size := len(input.RFQuestions); size == 0 || size > 30 {
		ctx.StatusCode(http.StatusBadRequest)
		fmt.Fprintln(ctx, "input's RFQuestions size should between 0 ~ 30")
		return
	}
	rfQuestions, err := FilterRFQuestions(util.GetAppID(ctx), input.RFQuestions)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		fmt.Fprintf(ctx, "Filter RFQuestions error, %vs", err)
		return
	}
	stdQuestions, err := FilterQuestions(util.GetAppID(ctx), rfQuestions)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		fmt.Fprintf(ctx, "Filter stdQuestions error, %vs", err)
		return
	}
	var response = struct {
		RFQuestions []struct {
			Content    string `json:"content"`
			CategoryID int64  `json:"CategoryId"`
			IsValid    *bool  `json:"isValid"`
		} `json:"RFQuestions"`
	}{}
	//Create dicts for lookup
	var stdQDict = make(map[string]StdQuestion, len(stdQuestions))
	for _, q := range stdQuestions {
		stdQDict[q.Content] = q
	}
	var RFQDict = make(map[string]bool, len(rfQuestions))
	for _, key := range rfQuestions {
		RFQDict[key] = true
	}

	//create response
	for _, q := range input.RFQuestions {
		var output = struct {
			Content    string `json:"content"`
			CategoryID int64  `json:"CategoryId"`
			IsValid    *bool  `json:"isValid"`
		}{}
		stdQ, exists := stdQDict[q]
		//IsValid should be true or false only at this time
		output.IsValid = &exists
		if exists {
			output.CategoryID = int64(stdQ.CategoryID)
		}
		//NotInRF need to be check for the null case.
		if _, inRF := RFQDict[q]; !inRF {
			output.IsValid = nil
		}

		output.Content = q
		response.RFQuestions = append(response.RFQuestions, output)
	}
	_, err = ctx.JSON(response)
	if err != nil {
		util.LogError.Printf("response json format failed, %v\n", err)
	}

}
