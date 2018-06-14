package BF

import "time"

type CmdTarget int

const (
	TargetStandardQuestion CmdTarget = iota
	TargetAnswer
	TargetMax
)

func (CmdTarget) Max() int {
	return int(TargetMax) - 1
}

type ResponseType int

const (
	TypeReplace ResponseType = iota
	TypeAppendFront
	TypeAppendBehind
	TypeAppendMax
)

func (ResponseType) Max() int {
	return int(TypeAppendMax) - 1
}

type CmdContent struct {
	// Type only allow keyword or regex
	Type  string   `json:"type"`
	Value []string `json:"value"`
}

type Cmd struct {
	ID        int           `json:"id"`
	Name      string        `json:"name"`
	Target    CmdTarget     `json:"target"`
	Rule      []*CmdContent `json:"rule"`
	Answer    string        `json:"answer"`
	Type      ResponseType  `json:"response_type"`
	Status    bool          `json:"status"`
	Begin     *time.Time    `json:"begin_time"`
	End       *time.Time    `json:"end_time"`
	LinkLabel []string      `json:"labels"`
}

func (r CmdContent) IsValid() bool {
	return (r.Type == "keyword" || r.Type == "regex") && len(r.Value) > 0
}

type Label struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	CreateTime time.Time `json:"createtime"`
	SSMID      string    `json:"label"`
	CmdCount   int       `json:"cmd_count"`
}

type CmdClass struct {
	ID       int         `json:"cid"`
	Name     string      `json:"name"`
	Cmds     []*Cmd      `json:"cmds"`
	Children []*CmdClass `json:"children"`
}