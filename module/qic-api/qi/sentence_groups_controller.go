package qi

import (
	autil "emotibot.com/emotigo/module/admin-api/util"
	"emotibot.com/emotigo/module/admin-api/util/requestheader"
	"emotibot.com/emotigo/module/qic-api/model/v1"
	"emotibot.com/emotigo/module/qic-api/util"
	"emotibot.com/emotigo/pkg/logger"
	"net/http"
)

var roleMapping map[string]int = map[string]int{
	"staff":    0,
	"customer": 1,
}

var positionMap map[string]int = map[string]int{
	"top":    0,
	"bottom": 1,
}

var roleCodeMap map[int]string = map[int]string{
	0: "staff",
	1: "customer",
}

var positionCodeMap map[int]string = map[int]string{
	0: "top",
	1: "bottom",
}

type SetenceGroupsResponse struct {
	Paging *util.Paging              `json:"paging"`
	Data   []SentenceGroupInResponse `json:"data"`
}

func sentenceGroupInReqToSentenceGroup(sentenceGroupInReq *SentenceGroupInReq) (group *model.SentenceGroup) {
	group = &model.SentenceGroup{
		Name: sentenceGroupInReq.Name,
	}

	sentences := []model.SimpleSentence{}
	for _, sid := range sentenceGroupInReq.Sentences {
		sentence := model.SimpleSentence{
			UUID: sid,
		}
		sentences = append(sentences, sentence)
	}
	group.Sentences = sentences

	if roleCode, ok := roleMapping[sentenceGroupInReq.Role]; ok {
		group.Role = roleCode
	} else {
		group.Role = -1
	}

	if poisitionCode, ok := positionMap[sentenceGroupInReq.Position]; ok {
		group.Position = poisitionCode
	} else {
		group.Position = -1
	}

	group.Distance = sentenceGroupInReq.PositionDistance
	return
}

func sentenceGroupToSentenceGroupInResponse(sg *model.SentenceGroup) (sgInRes SentenceGroupInResponse) {
	sgInRes = SentenceGroupInResponse{
		ID:               sg.UUID,
		Name:             sg.Name,
		Role:             roleCodeMap[sg.Role],
		Position:         positionCodeMap[sg.Position],
		PositionDistance: sg.Distance,
		Sentences:        sg.Sentences,
	}
	return
}

func handleCreateSentenceGroup(w http.ResponseWriter, r *http.Request) {
	enterprise := requestheader.GetEnterpriseID(r)

	groupInReq := SentenceGroupInReq{}
	err := autil.ReadJSON(r, &groupInReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	group := sentenceGroupInReqToSentenceGroup(&groupInReq)
	group.Enterprise = enterprise
	if group.Position == -1 || group.Role == -1 {
		http.Error(w, "bad sentence group", http.StatusBadRequest)
		return
	}

	createdGroup, err := CreateSentenceGroup(group)
	if err != nil {
		logger.Error.Printf("error while create sentence in handleCreateSentenceGroup, reason: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	groupInResponse := SentenceGroupInResponse{
		ID: createdGroup.UUID,
	}
	autil.WriteJSON(w, groupInResponse)
	return
}

func handleGetSentenceGroups(w http.ResponseWriter, r *http.Request) {
	enterprise := requestheader.GetEnterpriseID(r)
	filter := &model.SentenceGroupFilter{
		Limit:      0,
		Role:       -1,
		Position:   -1,
		Enterprise: enterprise,
	}

	total, groups, err := GetSentenceGroupsBy(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	groupsInRes := make([]SentenceGroupInResponse, len(groups))
	for idx, group := range groups {
		groupInRes := sentenceGroupToSentenceGroupInResponse(&group)
		groupsInRes[idx] = groupInRes
	}

	response := SetenceGroupsResponse{
		Paging: &util.Paging{
			Total: total,
			Page:  0,
			Limit: len(groups),
		},
		Data: groupsInRes,
	}

	autil.WriteJSON(w, response)
}

func handleGetSentenceGroup(w http.ResponseWriter, r *http.Request) {
	enterprise := requestheader.GetEnterpriseID(r)
	id := parseID(r)

	filter := &model.SentenceGroupFilter{
		UUID: []string{
			id,
		},
		Enterprise: enterprise,
		Role:       -1,
		Position:   -1,
		Limit:      0,
	}

	total, groups, err := GetSentenceGroupsBy(filter)
	if err != nil {
		logger.Error.Printf("err: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if total == 0 {
		http.NotFound(w, r)
		return
	}

	group := groups[0]

	groupInRes := sentenceGroupToSentenceGroupInResponse(&group)
	autil.WriteJSON(w, groupInRes)
}

func handleUpdateSentenceGroup(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)
	enterprise := requestheader.GetEnterpriseID(r)

	groupInReq := SentenceGroupInReq{}
	err := autil.ReadJSON(r, &groupInReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	group := sentenceGroupInReqToSentenceGroup(&groupInReq)
	group.Enterprise = enterprise

	updatedGroup, err := UpdateSentenceGroup(id, group)
	if err != nil {
		logger.Error.Printf("error while update sentence group in handleUpdateSentenceGroup, reason: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	groupInRes := sentenceGroupToSentenceGroupInResponse(updatedGroup)
	autil.WriteJSON(w, groupInRes)
	return
}

func handleDeleteSentenceGroup(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)

	err := DeleteSentenceGroup(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
