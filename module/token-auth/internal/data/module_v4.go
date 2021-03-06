package data

type ModuleDetailV4 struct {
	ID         int               `json:"id"`
	ParentId   int               `json:"parent_id"`
	ParentCmd  string            `json:"parent_cmd"`
	Code       string            `json:"code"`
	Cmd        string            `json:"cmd"`
	CmdKey     string            `json:"cmd_key"`
	Name       string            `json:"name"`
	Sort       int               `json:"sort"`
	Position   string            `json:"position"`
	Icon       string            `json:"icon"`
	Route      string            `json:"route"`
	IsLink     bool              `json:"is_link"`
	IsShow     bool              `json:"is_show"`
	CreateTime string            `json:"create_time"`
	SubCmd     []*ModuleDetailV4 `json:"sub_cmd"`
}
