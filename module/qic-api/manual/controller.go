package manual

import (
	"emotibot.com/emotigo/module/admin-api/util"
	"emotibot.com/emotigo/module/admin-api/util/requestheader"
	"emotibot.com/emotigo/module/qic-api/model/v1"
	"emotibot.com/emotigo/pkg/logger"
	"net/http"
)

type CallTimeRange struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
}

type Sampling struct {
	Percentage int `json:"percentage"`
	ByPerson   int `json:"byperson"`
}

type InspectTaskInReq struct {
	Name         string        `json:"task_name"`
	TimeRange    CallTimeRange `json:"call_time_range"`
	Outlines     []int64       `json:"outline_ids"`
	Staffs       []string      `json:"staff_ids"`
	Form         int64         `json:"form_id"`
	Sampling     Sampling      `json:"sampling_rule"`
	IsInspecting int8          `json:"is_inspecting"`
}

func inspectTaskInReqToInspectTask(inreq *InspectTaskInReq) (task *model.InspectTask) {
	taskOutlines := make([]model.Outline, len(inreq.Outlines))
	for idx := range inreq.Outlines {
		taskOutlines[idx] = model.Outline{
			ID: inreq.Outlines[idx],
		}
	}

	task = &model.InspectTask{
		Name:             inreq.Name,
		Outlines:         taskOutlines,
		Staffs:           inreq.Staffs,
		ExcludeInspected: inreq.IsInspecting,
		Form: model.ScoreForm{
			ID: inreq.Form,
		},
		InspectPercentage: inreq.Sampling.Percentage,
		InspectByPerson:   inreq.Sampling.ByPerson,
		CallStart:         inreq.TimeRange.StartTime,
		CallEnd:           inreq.TimeRange.EndTime,
	}
	return
}

func handleCreateTask(w http.ResponseWriter, r *http.Request) {
	inreq := InspectTaskInReq{}
	err := util.ReadJSON(r, &inreq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task := inspectTaskInReqToInspectTask(&inreq)
	task.Enterprise = requestheader.GetEnterpriseID(r)

	uuid, err := CreateTask(task)
	if err != nil {
		logger.Error.Printf("error while create inspect task in handleCreateTask, reason: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type Response struct {
		UUID string `json:"task_id"`
	}

	response := Response{
		UUID: uuid,
	}

	util.WriteJSON(w, response)
}