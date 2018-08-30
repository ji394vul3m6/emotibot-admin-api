package clustering

import "time"

const (
	//START_TIME input parameter, search starting time in UTC time format
	START_TIME = "start_time"
	//END_TIME input parameter, search end time in UTC time format
	END_TIME = "end_time"
)

//status code of clustering
const (
	S_PANIC = -1 + iota
	S_PROCESSING
	S_SUCCESS
)

type clusterTable struct {
	feedback      feedbackProps
	clusterTag    tagProps
	clusterResult resultProps
	report        reportProps
}

type tagProps struct {
	name         string
	id           string
	reportID     string
	clusteringID string
	tag          string
}

type resultProps struct {
	name       string
	id         string
	reportID   string
	feedbackID string
	clusterID  string
}

type reportProps struct {
	name        string
	id          string
	createdTime string
	startTime   string
	endTime     string
	status      string
	appid       string
	rType       string
}

type feedbackProps struct {
	name        string
	id          string
	question    string
	stdQuestion string
	createdTime string
	updatedTime string
	appid       string
	qType       string
}

//table properties name in database
var TableProps = clusterTable{
	feedback:      feedbackProps{name: "user_feedback", id: "id", question: "question", stdQuestion: "std_question", createdTime: "created_time", updatedTime: "updated_time", appid: "appid", qType: "type"},
	clusterTag:    tagProps{name: "clustering_tag", id: "id", reportID: "report_id", clusteringID: "clustering_id", tag: "tag"},
	clusterResult: resultProps{name: "clustering_result", id: "id", reportID: "unresolved_report_id", feedbackID: "feedback_id", clusterID: "cluster_id"},
	report:        reportProps{name: "unresolved_report", id: "id", createdTime: "created_time", startTime: "start_time", endTime: "end_time", status: "status", appid: "appid", rType: "type"},
}

// Report represent a clustering task
type Report struct {
	ID               uint64    `json:"id"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	ClusterSize      int       `json:"clusterSize"`
	UserQuestionSize int       `json:"userQuestionSize"`
	Status           int       `json:"status"`
}

// Cluster is a subset of Report, contains userQuestions as a group
type Cluster struct {
	ID               int      `json:"id"`
	UserQuestionSize int      `json:"userQuestionSize"`
	Tags             []string `json:"tags"`
}

// UserQuestion is user's unsolved question
type UserQuestion struct {
	ID          uint64    `json:"id"`
	Question    string    `json:"question"`
	StdQuestion string    `json:"std_question"`
	CreatedTime time.Time `json:"created_time"`
	UpdatedTime time.Time `json:"updated_time"`
}

type clusteringResult struct {
	numClustered int
	clusters     []clustering
	reportID     uint64
}

type clustering struct {
	feedbackID []uint64
	tags       []string
}

type rankerElm struct {
	idx     int
	avgDist float64
}

type ranker []rankerElm

func (r ranker) Len() int {
	return len(r)
}

func (r ranker) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r ranker) Less(i, j int) bool {
	return r[i].avgDist < r[j].avgDist
}

//StoreCluster store the cluster result
type StoreCluster interface {
	Store(cr *clusteringResult) error
}

//column name of <appid>_question
const (
	NQuestionID         = "Question_Id"
	NContent            = "Content"
	QuestionTableFormat = "%s_question"
)

//parameter name
const (
	PType = "type"
)

func isValidType(pType int) bool {

	switch pType {
	case 0:
		fallthrough
	case 1:
		return true
	default:
		return false
	}
}

//ReportQuery is a complex condition for querying reports
type ReportQuery struct {
	Reports   []string `json:"reports"`
	StartTime *int64   `json:"start_time"`
	EndTime   *int64   `json:"end_time"`
	UserID    *string  `json:"user_id"`
}
