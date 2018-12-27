package qi

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/mux"

	autil "emotibot.com/emotigo/module/admin-api/util"
	"emotibot.com/emotigo/module/qic-api/model/v1"
	"emotibot.com/emotigo/module/qic-api/util"
	"emotibot.com/emotigo/pkg/logger"
)

func handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	group := model.GroupWCond{}
	err := autil.ReadJSON(r, &group)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	createdGroup, err := CreateGroup(&group)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	autil.WriteJSON(w, createdGroup)
}

func handleGetGroups(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	filter, err := parseGroupFilter(&values)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	total, groups, err := GetGroupsByFilter(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	simpleGroups := make([]model.SimpleGroup, len(groups), len(groups))
	for i, group := range groups {
		simpleGroup := model.SimpleGroup{
			ID:   group.UUID,
			Name: group.Name,
		}

		simpleGroups[i] = simpleGroup
	}

	response := SimpleGroupsResponse{
		Paging: &util.Paging{
			Page:  filter.Page,
			Limit: filter.Limit,
			Total: total,
		},
		Data: simpleGroups,
	}

	autil.WriteJSON(w, response)
}

func parseID(r *http.Request) (id string) {
	vars := mux.Vars(r)
	return vars["id"]
}

func handleGetGroup(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)

	group, err := GetGroupBy(id)
	if err != nil {
		logger.Error.Printf("error while get group in handleGetGroup, reason: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if group == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	autil.WriteJSON(w, group)

}

func handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)

	group := model.GroupWCond{}
	err := autil.ReadJSON(r, &group)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = UpdateGroup(id, &group)
	if err != nil {
		logger.Error.Printf("error while update group in handleUpdateGroup, reason: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)

	err := DeleteGroup(id)

	if err != nil {
		logger.Error.Printf("error while delete group in handleDeleteGroup, reason: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	total, groups, err := GetGroupsByFilter(filter)
	if err != nil {
		logger.Error.Printf("error while get groups by filter in handleGetGroupsByFilter, reason: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := GroupsResponse{
		Paging: &util.Paging{
			Page:  filter.Page,
			Total: total,
			Limit: filter.Limit,
		},
		Data: groups,
	}

	autil.WriteJSON(w, response)
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

	dealStr := values.Get("deal")
	if dealStr != "" {
		filter.Deal, err = strconv.Atoi(dealStr)
		if err != nil {
			return
		}
	} else {
		filter.Deal = -1
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
		filter.Limit = 10
	}
	return
}
