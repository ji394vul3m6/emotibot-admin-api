package Stats

import (
	"errors"
	"time"

	"emotibot.com/emotigo/module/vipshop-admin/ApiError"
)

func GetAuditList(appid string, input *AuditInput) (*AuditRet, int, error) {
	list, totalCnt, err := getAuditListPage(appid, input, input.Page, input.ListPerPage)
	if err != nil {
		return nil, ApiError.DB_ERROR, err
	}

	ret := &AuditRet{
		TotalCount: totalCnt,
		Data:       list,
	}

	return ret, ApiError.SUCCESS, nil
}

func GetQuestionStatisticResult(appid string, day int, qType string) (*StatRet, int, error) {
	end := time.Now().Unix()
	start := end - int64(day*24*60*60)

	var data []*StatRow
	var err error
	if qType == "unsolved" {
		data, err = getUnresolveQuestionsStatistic(appid, start, end)
	} else {
		return nil, ApiError.REQUEST_ERROR, errors.New("Unsupport type")
	}

	if err != nil {
		return nil, ApiError.DB_ERROR, err
	}

	ret := StatRet{
		Data: data,
	}

	return &ret, ApiError.SUCCESS, nil
}
