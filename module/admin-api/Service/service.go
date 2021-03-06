package Service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"emotibot.com/emotigo/module/admin-api/util/AdminErrors"

	"emotibot.com/emotigo/module/admin-api/util"
	"emotibot.com/emotigo/pkg/logger"
	"emotibot.com/emotigo/pkg/misc/match"
)

var (
	serviceNLUKey     = "NLU"
	serviceSolrETLKey = "SOLRETL"
	cache             = map[string]*NLUResult{}

	matcher           = map[string]*match.Matcher{}
	matcherType       = match.FuzzyMode
	stdQuestionExpire = map[string]int64{}
)

const (
	dftExpirePeriod = 300
)

func GetNLUResult(appid string, sentence string) (*NLUResult, error) {
	if _, ok := cache[sentence]; ok {
		return cache[sentence], nil
	}

	url := strings.TrimSpace(getEnvironment(serviceNLUKey))
	if url == "" {
		return nil, errors.New("NLU Service not set")
	}
	param := map[string]string{
		"f":     "segment,sentenceType,keyword",
		"appid": appid,
		"q":     sentence,
	}
	body, err := util.HTTPGet(url, param, 30)
	if err != nil {
		return nil, err
	}

	nluResult := []NLUResult{}
	err = json.Unmarshal([]byte(body), &nluResult)
	if err != nil {
		return nil, err
	}
	if len(nluResult) < 1 {
		return nil, errors.New("No result")
	}
	cache[sentence] = &nluResult[0]
	return &nluResult[0], nil
}

func BatchGetNLUResults(appid string, sentences []string) (map[string]*NLUResult, error) {
	sentencePerReq := 20
	if len(sentences) < sentencePerReq {
		return GetNLUResults(appid, sentences)
	}

	dataChan := make(chan []string)
	resultsChan := make(chan (*map[string]*NLUResult))
	allResult := map[string]*NLUResult{}
	defer func() {
		close(dataChan)
		close(resultsChan)
	}()

	maxWorker := 5
	for idx := 0; idx < maxWorker; idx++ {
		go func(workNo int) {
			for {
				sentencesGroup, ok := <-dataChan
				// if channel close, just exit
				if !ok {
					return
				}
				logger.Trace.Printf("Worker %d receive %d sentences\n", workNo, len(sentencesGroup))
				ret, err := GetNLUResults(appid, sentencesGroup)
				if err != nil {
					logger.Error.Println("Get NLU Result error:", err.Error())
					ret = map[string]*NLUResult{}
				}
				logger.Trace.Printf("Worker %d finish query NLU\n", workNo)
				resultsChan <- &ret
			}
		}(idx)
	}

	packetNum := (len(sentences)-1)/sentencePerReq + 1
	go func() {
		for idx := 0; idx < len(sentences); idx += sentencePerReq {
			ending := idx + sentencePerReq
			if ending > len(sentences) {
				ending = len(sentences)
			}
			logger.Trace.Printf("Send sentence %d-%d into channel\n", idx, ending)
			dataChan <- sentences[idx:ending]
		}
		logger.Trace.Printf("Send %d packets into channel\n", packetNum)
	}()

	for packetNum > 0 {
		groupResult := <-resultsChan
		packetNum--
		if groupResult != nil {
			logger.Trace.Printf("Master get %d results\n", len(*groupResult))
		}
		for k, v := range *groupResult {
			allResult[k] = v
		}
	}

	return allResult, nil
}

func GetNLUResults(appid string, sentences []string) (map[string]*NLUResult, error) {
	url := strings.TrimSpace(getEnvironment(serviceNLUKey))
	if url == "" {
		return nil, errors.New("NLU Service not set")
	}
	param := map[string]interface{}{
		"flags":   "segment,sentenceType,keyword",
		"appid":   appid,
		"queries": sentences,
	}
	body, err := util.HTTPPostJSON(url, param, 30)
	// body, err := util.HTTPGet(url, param, 30)
	if err != nil {
		return nil, err
	}

	nluResult := []*NLUResult{}
	err = json.Unmarshal([]byte(body), &nluResult)
	if err != nil {
		return nil, err
	}
	if len(nluResult) < 1 {
		return nil, errors.New("No result")
	}

	ret := map[string]*NLUResult{}
	for idx, result := range nluResult {
		ret[result.Sentence] = nluResult[idx]
	}
	return ret, nil
}

func IncrementAddSolr(content []byte) (string, error) {
	url := getSolrIncrementURL()
	if url == "" {
		return "", errors.New("Solr-etl Service not set")
	}

	reader := bytes.NewReader(content)
	status, body, err := util.HTTPPostFileWithStatus(url, reader, "robot_manual_tagging.json", "file", 30)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return body, fmt.Errorf("Status not 200, is %d", status)
	}
	return body, nil
}

func DeleteInSolr(typeInSolr string, deleteSolrIDs map[string][]string) (string, error) {
	url := getSolrDeleteURL()
	for appid := range deleteSolrIDs {
		params := map[string]string{
			"ids":   strings.Join(deleteSolrIDs[appid], ","),
			"appid": appid,
			"type":  typeInSolr,
		}
		content, err := util.HTTPGet(url, params, 30)
		if err != nil {
			return "", err
		}
		logger.Trace.Println("Send to solr-etl: ", params)
		logger.Trace.Println("Get from delete in solr: ", content)
	}
	return "", nil
}

func getSolrIncrementURL() string {
	url := strings.TrimSpace(getEnvironment(serviceSolrETLKey))
	return fmt.Sprintf("%s/editorialincre", url)
}
func getSolrDeleteURL() string {
	url := strings.TrimSpace(getEnvironment(serviceSolrETLKey))
	return fmt.Sprintf("%s/editorial/deletebyids", url)
}
func getSolrDeleteByFieldURL() string {
	url := strings.TrimSpace(getEnvironment(serviceSolrETLKey))
	return fmt.Sprintf("%s/editorial/deletebyquery", url)
}

func getEnvironments() map[string]string {
	return util.GetEnvOf(ModuleInfo.ModuleName)
}

func getEnvironment(key string) string {
	envs := util.GetEnvOf(ModuleInfo.ModuleName)
	if envs != nil {
		if val, ok := envs[key]; ok {
			return val
		}
	}
	return ""
}

func GetRecommandStdQuestion(appid string, pattern string, n int) ([]string, AdminErrors.AdminError) {
	now := time.Now().Unix()
	typeStr := getEnvironment("MATCH_TYPE")
	if typeStr == "prefix" {
		matcherType = match.PrefixMode
	}
	expirePeriod, err := strconv.ParseInt(getEnvironment("MATCH_CACHE_TIMEOUT"), 10, 64)
	if err != nil || expirePeriod == 0 {
		expirePeriod = dftExpirePeriod
	}
	if now >= stdQuestionExpire[appid] || matcher[appid] == nil {
		questions, err := dalClient.Questions(appid)
		if err != nil {
			return nil, AdminErrors.New(AdminErrors.ErrnoAPIError, err.Error())
		}
		matcher[appid] = match.New(questions, matcherType)
		stdQuestionExpire[appid] = now + expirePeriod
		logger.Trace.Printf("Reload %d std questions of %s, timeout when %d\n", len(questions), appid, stdQuestionExpire[appid])
	}
	return matcher[appid].FindNSentence(pattern, n), nil
}

func GetRecommandStdQuestionFromDac(appid string, pattern string, n int) ([]string, AdminErrors.AdminError) {
	now := time.Now().Unix()
	typeStr := getEnvironment("MATCH_TYPE")
	if typeStr == "prefix" {
		matcherType = match.PrefixMode
	}
	expirePeriod, err := strconv.ParseInt(getEnvironment("MATCH_CACHE_TIMEOUT"), 10, 64)
	if err != nil || expirePeriod == 0 {
		expirePeriod = dftExpirePeriod
	}
	if now >= stdQuestionExpire[appid] || matcher[appid] == nil {
		questions, err := dacClient.Questions(appid)
		if err != nil {
			return nil, AdminErrors.New(AdminErrors.ErrnoAPIError, err.Error())
		}
		matcher[appid] = match.New(questions, matcherType)
		stdQuestionExpire[appid] = now + expirePeriod
		logger.Trace.Printf("Reload %d std questions of %s, timeout when %d\n", len(questions), appid, stdQuestionExpire[appid])
	}
	return matcher[appid].FindNSentence(pattern, n), nil
}


func IncrementAddSolrByType(content []byte, kind string) (string, error) {
	url := getSolrIncrementURL()
	if url == "" {
		return "", errors.New("Solr-etl Service not set")
	}
	filename := "other" + "_tagging.json"
	if kind == "other"{
		filename = "other" + "_tagging.json"
	}

	reader := bytes.NewReader(content)
	status, body, err := util.HTTPPostFileWithStatus(url, reader, filename, "file", 30)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return body, fmt.Errorf("Status not 200, is %d", status)
	}
	return body, nil
}

func DeleteInSolrByFiled(field string, deleteQuery string) (string, error) {
	url := getSolrDeleteByFieldURL()

	params := map[string]string{
		"query":   deleteQuery,
		"field": field,
		"core": "3rd_core",
	}
	content, err := util.HTTPGet(url, params, 30)
	if err != nil {
		return "", err
	}
	logger.Trace.Println("Send to solr-etl: ", params)
	logger.Trace.Println("Get from delete in solr: ", content)

	return "", nil
}