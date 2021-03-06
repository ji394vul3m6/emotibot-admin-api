package v1

import (
	"time"

	"emotibot.com/emotigo/module/admin-api/ELKStats/data"
	"emotibot.com/emotigo/module/admin-api/ELKStats/data/common"
	"emotibot.com/emotigo/module/admin-api/util/localemsg"
)

type SessionsRequest struct {
	StartTime         int64    `json:"start_time"`
	EndTime           int64    `json:"end_time"`
	Platforms         []string `json:"platform,omitempty"`
	Sex               []string `json:"sex,omitempty"`
	UserID            *string  `json:"uid,omitempty"`
	RatingMax         *int64   `json:"rating_max,omitempty"`
	RatingMin         *int64   `json:"rating_min,omitempty"`
	FeedbackStartTime *int64   `json:"feedback_start_time,omitempty"`
	FeedbackEndTime   *int64   `json:"feedback_end_time,omitempty"`
	Feedback          *string  `json:"feedback,omitempty"`
	Page              *int64   `json:"page,omitempty"`
	Limit             *int64   `json:"limit,omitempty"`
	SessionID         *string  `json:"session_id,omitempty"`
}

type SessionsQuery struct {
	data.CommonQuery
	Platforms         []string
	Sex               []string
	UserID            *string
	RatingMax         *int64
	RatingMin         *int64
	FeedbackStartTime *time.Time
	FeedbackEndTime   *time.Time
	Feedback          *string
	From              int64
	Limit             int64
	SessionID         *string
}

type SessionsResponse struct {
	TableHeader []data.TableHeaderItem `json:"table_header"`
	Data        []*SessionsData        `json:"data"`
	Limit       int64                  `json:"limit"`
	Page        int64                  `json:"page"`
	TotalSize   int64                  `json:"total_size"`
}

type SessionsDataBase struct {
	SessionID      string `json:"session_id"`
	UserID         string `json:"user_id"`
	Rating         int64  `json:"rating"`
	Feedback       string `json:"feedback"`
	CustomFeedback string `json:"custom_feedback"`
}

type SessionsRawData struct {
	SessionsDataBase
	StartTime    int64                  `json:"start_time"`
	EndTime      int64                  `json:"end_time"`
	CustomInfo   map[string]interface{} `json:"custom_info"`
	FeedbackTime int64                  `json:"feedback_time"`
}

type SessionsCommon struct {
	SessionsDataBase
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	FeedbackTime string `json:"feedback_time"`
}

type SessionsData struct {
	SessionsCommon
	CustomInfo map[string]interface{} `json:"custom_info"`
}

type SessionsExportData struct {
	SessionsCommon
	CustomInfo string `json:"custom_info"`
}

var SessionsTableHeader = map[string][]data.TableHeaderItem{
	"zh-cn": []data.TableHeaderItem{
		data.TableHeaderItem{
			Text: "会话ID",
			ID:   common.SessionsMetricSessionID,
		},
		data.TableHeaderItem{
			Text: "会话开始时间",
			ID:   common.SessionsMetricStartTime,
		},
		data.TableHeaderItem{
			Text: "会话结束时间",
			ID:   common.SessionsMetricEndTime,
		},
		data.TableHeaderItem{
			Text: "用户ID",
			ID:   common.SessionsMetricUserID,
		},
		data.TableHeaderItem{
			Text: "满意度",
			ID:   common.SessionsMetricRating,
		},
		data.TableHeaderItem{
			Text: "反馈选择",
			ID:   common.SessionsMetricFeedback,
		},
		data.TableHeaderItem{
			Text: "反馈文字",
			ID:   common.SessionsMetricCustomFeedback,
		},
		data.TableHeaderItem{
			Text: "反馈时间",
			ID:   common.SessionsMetricFeedbackTime,
		},
	},
	"zh-tw": []data.TableHeaderItem{
		data.TableHeaderItem{
			Text: "會話ID",
			ID:   common.SessionsMetricSessionID,
		},
		data.TableHeaderItem{
			Text: "會話開始時間",
			ID:   common.SessionsMetricStartTime,
		},
		data.TableHeaderItem{
			Text: "會話結束時間",
			ID:   common.SessionsMetricEndTime,
		},
		data.TableHeaderItem{
			Text: "用戶ID",
			ID:   common.SessionsMetricUserID,
		},
		data.TableHeaderItem{
			Text: "滿意度",
			ID:   common.SessionsMetricRating,
		},
		data.TableHeaderItem{
			Text: "反饋選擇",
			ID:   common.SessionsMetricFeedback,
		},
		data.TableHeaderItem{
			Text: "反饋文字",
			ID:   common.SessionsMetricCustomFeedback,
		},
		data.TableHeaderItem{
			Text: "反饋時間",
			ID:   common.SessionsMetricFeedbackTime,
		},
	},
}

func GetSessionsExportHeader(locale string) []string {
	var SessionsExportHeader = []string{
		localemsg.Get(locale, "sessionId"),
		localemsg.Get(locale, "startTime"),
		localemsg.Get(locale, "endTime"),
		localemsg.Get(locale, "userId"),
		localemsg.Get(locale, "rating"),
		localemsg.Get(locale, "customInfo"),
		localemsg.Get(locale, "feedback"),
		localemsg.Get(locale, "customFeedback"),
		localemsg.Get(locale, "feedbackTime"),
	}
	return SessionsExportHeader
}

type SessionsExportResponse struct {
	ExportID string `json:"export_id"`
}

type SessionsExportStatusResponse struct {
	Status string `json:"status"`
}
