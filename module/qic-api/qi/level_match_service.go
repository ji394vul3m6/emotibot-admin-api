package qi

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"emotibot.com/emotigo/pkg/logger"

	model "emotibot.com/emotigo/module/qic-api/model/v1"
	"emotibot.com/emotigo/module/qic-api/sensitive"
	"emotibot.com/emotigo/module/qic-api/util/logicaccess"
)

//error message
var (
	ErrNoArgument         = errors.New("Need arguments")
	ErrTimeoutSet         = errors.New("timeout must be larger than zero")
	ErrWrongExpression    = errors.New("wrong conversation flow expression")
	ErrRequestNotEqualGet = errors.New("request level record not equals to get")
	ErrNoModels           = errors.New("No prediction model is available")
	ErrNoRuleGroupFound   = errors.New("No rule group is found")
)

//MatchedData stores the index of input and the matched ID (tag id) and its relative data
type MatchedData struct {
	Index   int
	Matched map[uint64]*logicaccess.AttrResult
	lock    sync.Mutex
}

//SenGroupCriteria is SentenceGroup matching criteria
type SenGroupCriteria struct {
	ID         uint64
	SentenceID []uint64
	Role       int
	Position   int
	Range      int
}

//ExprNode is used to transform expression to node struct
type ExprNode struct {
	withNot bool
	isThen  bool
	uuid    string
}

//ConFlowCriteria is conversation flow matching critera
type ConFlowCriteria struct {
	ID         uint64
	Repeat     int
	Expression string

	startMust bool
	nodes     []*ExprNode
}

//RuleCriteria is criteria for rule level
type RuleCriteria struct {
	ID     uint64
	Min    int
	Score  int
	Method int
	CFIDs  []uint64
}

//RuleMatchedResult is used to return for rule check result
type RuleMatchedResult struct {
	Valid bool
	Score int //plus or minus
}

type SentenceWithPrediction struct {
	ID    uint64 `json:"id"`
	Valid bool   `json:"valid"`
	model.SimpleSentence
	MatchedSegments []*model.SegmentMatch `json:"segment_predictions"`
	Credit          *SentenceCredit       `json:"-"`
}

type BothExceptionCredit struct {
	Staff    []*SentenceWithPrediction `json:"staff"`
	Customer []*SentenceWithPrediction `json:"customer"`
}

type StaffExceptionCredit struct {
	Staff []*SentenceWithPrediction `json:"staff"`
}
type CustomerExceptionCredit struct {
	Customer []*SentenceWithPrediction `json:"customer"`
}

type SilenceExceptionCredit struct {
	Before BothExceptionCredit  `json:"before"`
	After  StaffExceptionCredit `json:"after"`
}
type SegmentTimeRange struct {
	Start float64 `json:"start_time"`
	End   float64 `json:"end_time"`
}
type SilenceRuleCredit struct {
	ID          int64                  `json:"id"`
	Name        string                 `json:"name"`
	Valid       bool                   `json:"valid"`
	Score       int                    `json:"score"`
	CreditID    int64                  `json:"revise_id,string"`
	Revise      int                    `json:"revise"`
	Comment     string                 `json:"comment"`
	Setting     model.SilenceRule      `json:"setting"`
	Exception   SilenceExceptionCredit `json:"exception"`
	InvalidSegs []SegmentTimeRange     `json:"invalid_segment"`
}

type SpeedExceptionCredit struct {
	Under CustomerExceptionCredit `json:"under"`
	Over  CustomerExceptionCredit `json:"over"`
}

type SpeedRuleCredit struct {
	ID        int64                `json:"id"`
	Name      string               `json:"name"`
	Valid     bool                 `json:"valid"`
	CreditID  int64                `json:"revise_id,string"`
	Revise    int                  `json:"revise"`
	Comment   string               `json:"comment"`
	Score     int                  `json:"score"`
	Setting   model.SpeedRule      `json:"setting"`
	Exception SpeedExceptionCredit `json:"exception"`
}

type InterposalRuleCredit struct {
	ID          int64                `json:"id"`
	Name        string               `json:"name"`
	Valid       bool                 `json:"valid"`
	CreditID    int64                `json:"revise_id,string"`
	Revise      int                  `json:"revise"`
	Comment     string               `json:"comment"`
	Score       int                  `json:"score"`
	Setting     model.InterposalRule `json:"setting"`
	InvalidSegs []SegmentTimeRange   `json:"invalid_segment"`
}

//RuleGrpCredit is the result of the segments
type RuleGrpCredit struct {
	ID      uint64            `json:"id"`
	Plus    int               `json:"-"`
	Score   int               `json:"score"`
	Rules   []*RuleCredit     `json:"rules"`
	Setting *model.GroupWCond `json:"setting"`

	SilenceRule    []*SilenceRuleCredit    `json:"silence_rule"`
	SpeedRule      []*SpeedRuleCredit      `json:"speed_rule"`
	InterposalRule []*InterposalRuleCredit `json:"interposal_rule"`

	Matched []*MatchedData `json:"-"`
}

//RuleCredit stores the rule level result
type RuleCredit struct {
	ID       uint64                    `json:"id"`
	Valid    bool                      `json:"valid"`
	Score    int                       `json:"score"`
	CreditID int64                     `json:"revise_id,string"`
	Revise   int                       `json:"revise"`
	Comment  string                    `json:"comment"`
	CFs      []*ConversationFlowCredit `json:"conversation_flow"`
	Setting  *ConversationRuleInRes    `json:"setting"`
}

//ConversationFlowCredit stores the conversation flow level result
type ConversationFlowCredit struct {
	ID           uint64                 `json:"id"`
	Valid        bool                   `json:"valid"`
	SentenceGrps []*SentenceGrpCredit   `json:"sentence_groups"`
	Setting      *ConversationFlowInRes `json:"setting"`
}

//SentenceGrpCredit stores the sentence group level result
type SentenceGrpCredit struct {
	ID        uint64                   `json:"id"`
	Valid     bool                     `json:"valid"`
	Sentences []*SentenceCredit        `json:"sentences"`
	Setting   *SentenceGroupInResponse `json:"setting"`
}

//SentenceCredit stores the matched sentence and its relative tag information
type SentenceCredit struct {
	ID              uint64                `json:"id"`
	Valid           bool                  `json:"valid"`
	Segments        []int                 `json:"-"`
	Tags            []*TagCredit          `json:"-"`
	MatchedSegments []*model.SegmentMatch `json:"segment_predictions"`
	Setting         *DataSentence         `json:"setting"`
}

// TagCredit stores the matched tag information and the segment id
//	Match is the keyword that segment matched.
//	MatchTxt is the segment text that matched with the Match.
type TagCredit struct {
	ID         uint64
	Score      int
	Match      string
	MatchTxt   string
	SegmentIdx int
	SegmentID  int64 //for controller usage
}

/*
//SensitiveUsrCol is the usr column exception
type SensitiveUsrCol struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	InputName   string `json:"inputname"`
	Type        int    `json:"type"`
	Description string `json:"description"`
	//Values
}
*/
//SWExceptionMatched is the exception and its matched segment
type SWExceptionMatched struct {
	Customer  []*SentenceWithPrediction `json:"customer"`
	CustomCol []*sensitive.CustomValues `json:"customcol"`
	Staff     []*SentenceWithPrediction `json:"staff"`
}

//SWSettingException is the setting of sensitive word and exception matched segments
type SWSettingException struct {
	ID         string             `json:"sw_id"`
	Name       string             `json:"sw_name"`
	Score      int                `json:"score"`
	Exceptions SWExceptionMatched `json:"exception"`
}

//SWRuleCredit is the sensitive word result credit
type SWRuleCredit struct {
	Valid               bool               `json:"valid"`
	Score               int                `json:"score"`
	CreditID            int64              `json:"revise_id,string"`
	Revise              int                `json:"revise"`
	Comment             string             `json:"comment"`
	InvalidSegs         []int64            `json:"invalid_segment"`
	CustomMatched       bool               `json:"customcol_matched"`
	SettingAndException SWSettingException `json:"setting"`
}

//FlowExpressionToNode converts the conversation flow expression to node
func (c *ConFlowCriteria) FlowExpressionToNode() error {

	token := strings.Split(c.Expression, " ")
	numOfToken := len(token)
	queue := make(chan string, 99999)

	if numOfToken < 2 {
		return ErrWrongExpression
	}

	lToken := strings.ToLower(token[0])
	switch lToken {
	case "if":
	case "must":
		c.startMust = true
	default:
		return ErrWrongExpression
	}

	queue <- lToken

	for i := 1; i < numOfToken; i++ {

		lToken := strings.ToLower(token[i])
		switch lToken {
		case "not":
			queue <- lToken
		case "and":
			fallthrough
		case "then":
			if len(queue) != 0 {
				return ErrWrongExpression
			}
			queue <- lToken
		default:
			if len(queue) == 0 {
				return ErrWrongExpression
			}
			n := &ExprNode{}
			numOfQue := len(queue)
			for j := 0; j < numOfQue; j++ {
				last := <-queue
				switch last {
				case "and":
				case "not":
					n.withNot = !n.withNot
				case "then":
					n.isThen = true
				case "if":
				case "must":
				default:
					return ErrWrongExpression
				}
			}
			n.uuid = token[i]
			c.nodes = append(c.nodes, n)
		}

	}

	if len(queue) != 0 {
		return ErrWrongExpression
	}
	return nil
}

//SetData sets the data for thread-safe
func (m *MatchedData) SetData(d *logicaccess.AttrResult) {

	if d != nil && d.SentenceID > 0 {
		m.lock.Lock()
		m.Matched[d.Tag] = d
		m.lock.Unlock()
	}

}

//MatchedIdx stores which index of sentence matches the target id
type MatchedIdx struct {
	Index     []uint64
	MatchedID uint64
}

//Concurrency sets the number of goroutine used to call cu module
const (
	Concurrency = 1
	Threshold   = 60
)

func worker(ctx context.Context, target <-chan uint64, errChan chan<- error,
	segments []string, wg *sync.WaitGroup, collected []*MatchedData) {
	defer wg.Done()
	numOfData := len(collected) + 1
	for {
		select {
		case id, more := <-target:
			if !more {
				return
			}
			pr, err := BatchPredict(id, Threshold, segments)
			if err != nil {
				errChan <- err
				logger.Error.Printf("batch predict failed. %s\n", err)
				return
			}

			for i := 0; i < len(pr.Dialogue); i++ {
				v := pr.Dialogue[i]
				if v.SentenceID > 0 && v.SentenceID < numOfData {
					idx := v.SentenceID - 1
					collected[idx].SetData(&v)
				}
			}

			for i := 0; i < len(pr.Keyword); i++ {
				v := pr.Keyword[i]
				if v.SentenceID > 0 && v.SentenceID < numOfData {
					idx := v.SentenceID - 1
					collected[idx].SetData(&v)
				}
			}

			for i := 0; i < len(pr.UsrResponse); i++ {
				v := pr.UsrResponse[i]
				if v.SentenceID > 0 && v.SentenceID < numOfData {
					idx := v.SentenceID - 1
					collected[idx].SetData(&v)
				}
			}
		case <-ctx.Done():
			return
		}
	}

}

// TagMatch checks each segment for trained model.
// return value: a slice of matchData gives the each sentences and its matched tag and matched data
func TagMatch(modelIDs []uint64, segments []string, timeout time.Duration) ([]*MatchedData, error) {

	numOfCtx := len(segments)
	numOfModels := len(modelIDs)

	if numOfCtx == 0 || len(modelIDs) == 0 {
		return nil, ErrNoArgument
	}
	if timeout <= 0 {
		return nil, ErrTimeoutSet
	}

	//context and channel init
	var wg sync.WaitGroup
	wg.Add(Concurrency)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	target := make(chan uint64, numOfModels)
	defer cancel()

	//init the response structure
	matches := make([]*MatchedData, numOfCtx, numOfCtx)
	for i := 0; i < numOfCtx; i++ {
		matches[i] = &MatchedData{}
		matches[i].Matched = make(map[uint64]*logicaccess.AttrResult)
		matches[i].Index = i + 1
	}

	sort.Slice(modelIDs, func(i, j int) bool { return modelIDs[i] < modelIDs[j] })
	var lastTag uint64
	//start to input the target tag id
	for _, v := range modelIDs {
		//avoid the duplicate tag
		if lastTag != v {
			target <- v
			lastTag = v
		}
	}
	close(target)

	errChan := make(chan error, Concurrency)
	//call goroutine to do job concurrency
	for i := 0; i < Concurrency; i++ {
		go worker(ctx, target, errChan, segments, &wg, matches)
	}
	wg.Wait()

	var err error
	if len(errChan) > 0 {
		err = <-errChan
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}

	return matches, err
}

//SentencesMatch gives the sentence id that is matched by which segment index
//c is the argument map[sentence id][]tag id.
//senMatched is matched tag id for each segment
func SentencesMatch(senMatched []map[uint64]bool, c map[uint64][]uint64) (map[uint64][]int, error) {
	//func SentencesMatch(m []*MatchedData, c map[uint64][]uint64) (map[uint64][]int, error) {

	resp := make(map[uint64][]int, len(c))

	//for loop the criteria for each tag in each sentence
	for sID, tagIDs := range c {
		//compare the given matched tags in each segement to the criteria
		for idx, d := range senMatched {
			if len(d) > 0 {
				numOfChild := len(tagIDs)
				var count int
				//check whether this segment match all tags
				for _, tagID := range tagIDs {
					if _, ok := d[tagID]; !ok {
						break
					}
					count++
				}
				//count!=0 to avoid the unfinished sentences, set sentences with no tag to false
				if count != 0 && count == numOfChild {
					resp[sID] = append(resp[sID], idx+1)
				}
			}
		}
	}
	return resp, nil
}

//SentencesMatchWithZeroBasedIndex matches the sentence and gives the 0 based index
func SentencesMatchWithZeroBasedIndex(senMatched []map[uint64]bool, c map[uint64][]uint64) (map[uint64][]int, error) {
	//func SentencesMatch(m []*MatchedData, c map[uint64][]uint64) (map[uint64][]int, error) {

	resp := make(map[uint64][]int, len(c))

	//for loop the criteria for each tag in each sentence
	for sID, tagIDs := range c {
		//compare the given matched tags in each segement to the criteria
		for idx, d := range senMatched {
			if len(d) > 0 {
				numOfChild := len(tagIDs)
				var count int
				//check whether this segment match all tags
				for _, tagID := range tagIDs {
					if _, ok := d[tagID]; !ok {
						break
					}
					count++
				}
				//count!=0 to avoid the unfinished sentences, set sentences with no tag to false
				if count != 0 && count == numOfChild {
					resp[sID] = append(resp[sID], idx)
				}
			}
		}
	}
	return resp, nil
}

//SentenceGroupMatch matches the given matched sentence to sentence group
//matchedSen is matched sentence id and the matched segment id
//c is the sentenceGroup criteria used to judge whether the sentence group is meet
//segments is the segments data
//return value: the sentence group id which is meet by which segment index
func SentenceGroupMatch(matchedSen map[uint64][]int,
	c map[uint64]*SenGroupCriteria, segments []*SegmentWithSpeaker) (map[uint64][]int, error) {
	resp := make(map[uint64][]int, len(c))

	numOfSegs := len(segments)
	//for loop for each sentence group critera
	for _, criteria := range c {
		//for loop for each sentence in sentence group
		for _, sID := range criteria.SentenceID {

			//check whether one of the segment is meet the sentence
			if segIdxs, ok := matchedSen[sID]; ok {

				//record which segment meet the sentence
				for _, segIdx := range segIdxs {
					if segIdx-1 < numOfSegs && segments[segIdx-1] != nil {

						s := segments[segIdx-1]
						//check the role
						//if the role is any. < 0 for older version
						if criteria.Role == roleMapping["any"] || criteria.Role < 0 {
							resp[criteria.ID] = append(resp[criteria.ID], segIdx)
						} else if s.Speaker == criteria.Role {
							resp[criteria.ID] = append(resp[criteria.ID], segIdx)
							break
						}
					}

				}
			}
		}
	}

	return resp, nil
}

func extractTagMatchedData(tagMatchDat []*MatchedData) []map[uint64]bool {

	numOfData := len(tagMatchDat)
	resp := make([]map[uint64]bool, numOfData, numOfData)
	for i := 0; i < numOfData; i++ {
		resp[i] = make(map[uint64]bool)
		d := tagMatchDat[i]
		if d != nil {
			for tagID := range d.Matched {
				resp[i][tagID] = true
			}
		}
	}

	return resp

}

//ConversationFlowMatch checks one conversation at one time and give whether this conversation flow is meet or not
//parameters:
//matchSgID is the segment index for each sentence group that is matched
//senGrpCriteria is the criteria for each sentence group
//cfCriteria is the conversation flow criteria
//senGrpUUIDMapID is the map from uuid to id in sentence group
//totalSeg is the total lines in this user input context
func ConversationFlowMatch(matchSgID map[uint64][]int, senGrpCriteria map[uint64]*SenGroupCriteria,
	cfCriteria *ConFlowCriteria, senGrpUUIDMapID map[string]uint64, totalSeg int) (matched bool, err error) {

	if cfCriteria == nil {
		return
	}
	//empty the node in case for reuse
	cfCriteria.nodes = []*ExprNode{}
	cfCriteria.startMust = false
	//transform the expression to node struct
	err = cfCriteria.FlowExpressionToNode()
	if err != nil {
		logger.Error.Printf("Transform expresionn %s failed. %s\n", cfCriteria.Expression, err)
		return
	}

	//copy the matched sentence group id for later use
	copyMatchSgID := make(map[uint64][]int, len(matchSgID))
	for k, v := range matchSgID {
		s := make([]int, len(v))
		copy(s, v)
		copyMatchSgID[k] = s
	}

	for z := 0; z < cfCriteria.Repeat; z++ {
		var lastSegmentIdx int
		useFirstSeg := true
		//check for each sentence group which is already in order
		for idx, v := range cfCriteria.nodes {

			id, ok := senGrpUUIDMapID[v.uuid]
			if !ok {
				logger.Error.Printf("Cannot find uuid %s with its id\n", v.uuid)
				return
			}

			criteria, ok := senGrpCriteria[id]
			if !ok {
				logger.Error.Printf("Cannot find sentence group %d with its information\n", id)
				return
			}

			matchThisSenGrp := true
			numOfMatchedSeg := len(copyMatchSgID[id])

			if v.withNot {
				if numOfMatchedSeg != 0 {
					matchThisSenGrp = false
				}
			} else {
				if numOfMatchedSeg == 0 {
					matchThisSenGrp = false
				}
			}

			if idx == 0 {
				//no match at begining
				if !matchThisSenGrp {
					//starts with if
					if !cfCriteria.startMust {
						matched = true
					}
					return
				}

				switch criteria.Position {
				//must start in n words
				case 0:
					if copyMatchSgID[id][0] > criteria.Range {
						return
					}
				//must ends with this sentence group in the n last words
				case 1:
					if totalSeg-copyMatchSgID[id][numOfMatchedSeg-1] > criteria.Range {
						return
					}
					useFirstSeg = false
				//no assigned
				default:
				}
			} else {
				if !matchThisSenGrp {
					return
				}
				if v.withNot {
					continue
				}
				//check then scenario
				if criteria.Range > 0 {
					segIdx := copyMatchSgID[id][0]
					if segIdx-lastSegmentIdx > criteria.Range || segIdx < lastSegmentIdx {
						return
					}
				} else {
					if v.isThen {
						if copyMatchSgID[id][0] < lastSegmentIdx {
							return
						}
					}
				}
			}

			if useFirstSeg {
				lastSegmentIdx = copyMatchSgID[id][0]
				copyMatchSgID[id] = copyMatchSgID[id][1:]
			} else {
				lastSegmentIdx = copyMatchSgID[id][numOfMatchedSeg-1]
				copyMatchSgID[id] = copyMatchSgID[id][:numOfMatchedSeg]
			}
		}
	}
	matched = true
	return
}

//RuleMatch used to check whether the rule level meets. gives the map that the rule id meets the criterion and its plus score
//parameters:
//cfMatchID is the map recording the conversation flow id which meets the criterion
func RuleMatch(cfMatchID map[uint64]bool, criteria map[uint64]*RuleCriteria) (map[uint64]*RuleMatchedResult, int, error) {
	resp := make(map[uint64]*RuleMatchedResult, len(criteria))
	var totalScore int
	for ruleID, criterion := range criteria {
		var count int
		var matched bool
		var plus int
		for _, cfID := range criterion.CFIDs {
			if v, ok := cfMatchID[cfID]; ok && v {
				count++
			}
		}
		if count >= criterion.Min {
			matched = true
		}
		if criterion.Method == int(methodStringToCode["negative"]) {
			matched = !matched
		}

		if matched {
			if criterion.Score > 0 {
				plus = plus + criterion.Score
			}
		} else {
			if criterion.Score < 0 {
				plus = plus + criterion.Score
			}
		}
		totalScore += plus
		resp[ruleID] = &RuleMatchedResult{Valid: matched, Score: plus}
	}
	return resp, totalScore, nil
}

// RuleGroupCriteria gives the result of the criteria used to the group
// ruleGroup is the id of the rule group to validate.
// segments must be sorted by time ascended.
// timeout is used to wait for cu module result.
// if success, a RuleGrpCredit is returned.
func RuleGroupCriteria(ruleGroups []model.Group, segments []*SegmentWithSpeaker, timeout time.Duration) ([]*RuleGrpCredit, error) {
	numOfLines := len(segments)
	if numOfLines == 0 {
		return nil, ErrNoArgument
	}
	if len(ruleGroups) == 0 {
		return nil, ErrNoArgument
	}

	logger.Info.Printf("doing %+v rule group credit\n", ruleGroups)
	enterprise := ruleGroups[0].EnterpriseID
	models, err := GetUsingModelByEnterprise(enterprise)
	if err != nil {
		logger.Error.Printf("get the model failed. %s\n", err.Error())
		return nil, err
	}
	numOfModels := len(models)
	if numOfModels == 0 {
		return nil, ErrNoModels
	} else if numOfModels > 1 {
		logger.Warn.Printf("More than 1 models is marked as using status with enterprise %s\n", enterprise)
		logger.Warn.Printf("Using the first one %d as prediction model", models[0].ID)
	}

	//extract the words
	lines := make([]string, 0, numOfLines)
	for _, v := range segments {
		if v != nil {
			lines = append(lines, v.Text)
		}
	}

	//--------------------------------------------------------------------------
	//do the checking, tag match
	tagMatchDat, err := TagMatch([]uint64{models[0].ID}, lines, timeout)
	if err != nil {
		return nil, fmt.Errorf("tag match failed, %v", err)
	}
	if len(tagMatchDat) != numOfLines {
		return nil, fmt.Errorf("get less tag match sentence %d with %d", len(tagMatchDat), numOfLines)
	}

	resp := make([]*RuleGrpCredit, 0, len(ruleGroups))
	for _, ruleGroup := range ruleGroups {
		//get the relation table from RuleGroup to Tag
		levels, _, err := GetLevelsRel(LevRuleGroup, LevTag, []uint64{uint64(ruleGroup.ID)}, true)
		if err != nil {
			logger.Error.Printf("get level relations failed. %s\n", err)
			return nil, err
		}

		//check the return level
		//If the level is not the same, it might mean data corruption.
		tagLev := int(LevTag)
		if len(levels) < tagLev {
			logger.Error.Printf("get less relation table. %d\n", tagLev)
			return nil, errors.New("get less relation table")
		}

		numOfSens := len(levels[LevSentence])
		//sentence(句子)
		sentenceCreditMap := make(map[uint64]*SentenceCredit)
		//extract the sentence id and tag id
		senIDs := make([]uint64, 0, numOfSens)
		tagIDs := make([]uint64, 0, numOfSens)
		for sID, tIDs := range levels[LevSentence] {
			senIDs = append(senIDs, sID)
			tagIDs = append(tagIDs, tIDs...)

			credit := &SentenceCredit{ID: sID}
			sentenceCreditMap[sID] = credit
		}

		//--------------------------------------------------------------------------
		//do the sentence check
		// segMatchedTag struct []map[uint64]bool,
		// every slice index is a segment, which has a bunch of uint64(tag id),
		// use map for quick search later
		segMatchedTag := extractTagMatchedData(tagMatchDat)
		//do the checking, sentence match
		senMatchDat, err := SentencesMatch(segMatchedTag, levels[LevSentence])
		if err != nil {
			logger.Warn.Printf("doing sentence  match failed.%s\n", err)
			return nil, err
		}

		//stores the sentence result in map for later user

		for senID, segIdxs := range senMatchDat {
			var credit *SentenceCredit

			if v, ok := sentenceCreditMap[senID]; ok {
				credit = v
				v.Valid = true
			} else {
				logger.Error.Printf("sentence matched id %d exist in credit map, check the relation sentence to tag\n", senID)
				return nil, ErrRequestNotEqualGet
			}

			for _, segIdx := range segIdxs {
				// because slice is 0 based index, but cu is 1 based index.
				matched := tagMatchDat[segIdx-1]
				for _, data := range matched.Matched {
					var tagCredit TagCredit
					//TagID
					tagCredit.ID = data.Tag
					tagCredit.Score = data.Score
					//SentenceID is the cu term for segment Idx, which is 1 based index
					tagCredit.SegmentIdx = data.SentenceID
					tagCredit.Match = data.Match
					tagCredit.MatchTxt = data.MatchText
					tagCredit.SegmentID = segments[segIdx-1].ID
					credit.Tags = append(credit.Tags, &tagCredit)
				}
			}
		}

		//--------------------------------------------------------------------------

		//extract the sentence group id
		conContainSenGrp := levels[LevConversation]
		senGrpIDs := make([]uint64, 0)
		cfIDs := make([]uint64, 0, len(conContainSenGrp))
		for cfID, senGrpIDList := range conContainSenGrp {
			senGrpIDs = append(senGrpIDs, senGrpIDList...)
			cfIDs = append(cfIDs, cfID)
		}

		//get the sentence group information for condition usage
		sgFilter := &model.SentenceGroupFilter{ID: senGrpIDs}
		_, senGrp, err := GetSentenceGroupsBy(sgFilter)

		if err != nil {
			logger.Error.Printf("get sentence group info failed.%s\n", err)
			return nil, err
		}
		numOfSenGrp := len(senGrp)
		//may duplicate
		/*
			if numOfSenGrp != len(senGrpIDs) {
				logger.Error.Printf("request sentence group(%d) %v not equal to get %d\n", len(senGrpIDs), senGrpIDs, numOfSenGrp)
				return nil, ErrRequestNotEqualGet
			}
		*/

		//transform the sentence group information to the sentence group critera struct
		senGrpContainSen := levels[LevSenGroup]
		senGrpCriteria := make(map[uint64]*SenGroupCriteria)
		senGrpUUIDMapID := make(map[string]uint64, numOfSenGrp)
		senGrpCreditMap := make(map[uint64]*SentenceGrpCredit)

		for i := 0; i < numOfSenGrp; i++ {
			id := uint64(senGrp[i].ID)
			var criterion SenGroupCriteria
			credit := &SentenceGrpCredit{ID: id}
			senGrpCreditMap[id] = credit
			if senIDs, ok := senGrpContainSen[id]; ok {
				senGrpCriteria[id] = &criterion
				senGrpCriteria[id].ID = id
				senGrpCriteria[id].Role = senGrp[i].Role
				senGrpCriteria[id].Range = senGrp[i].Distance
				senGrpCriteria[id].Position = senGrp[i].Position
				senGrpCriteria[id].SentenceID = senIDs
				senGrpUUIDMapID[senGrp[i].UUID] = id
			} else {
				logger.Error.Printf("No sentence group id %d in sentence group table, but exist in relation table\n", id)
				return nil, ErrRequestNotEqualGet
			}
		}

		//do the check, sentence group
		matchSgID, err := SentenceGroupMatch(senMatchDat, senGrpCriteria, segments)
		if err != nil {
			logger.Warn.Printf("doing sentence group match failed.%s\n", err)
			return nil, err
		}

		//stores the sentence group level result
		for senGrp, sentences := range senGrpContainSen {
			for _, sID := range sentences {
				if sCredit, ok := sentenceCreditMap[sID]; ok {
					if _, ok := matchSgID[senGrp]; ok {
						senGrpCreditMap[senGrp].Valid = true
					}
					senGrpCreditMap[senGrp].Sentences = append(senGrpCreditMap[senGrp].Sentences, sCredit)
				} else {
					logger.Error.Printf("sentence matched id %d doesn't exitst in credit map\n", sID)
					return nil, ErrRequestNotEqualGet
				}
			}

		}
		//--------------------------------------------------------------------------

		//get the conversation flow inforamtion
		cfFilter := &model.ConversationFlowFilter{ID: cfIDs}
		_, cfInfo, err := GetConversationFlowsBy(cfFilter)
		if err != nil {
			logger.Error.Printf("get conversation flow failed.%s\n", err)
			return nil, err
		}

		//sorting the matched segment index
		for _, segIdxs := range matchSgID {
			sort.Ints(segIdxs)
		}

		matchCFID := make(map[uint64]bool)
		cfCreditMap := make(map[uint64]*ConversationFlowCredit)
		//doing check for each conversation flow
		for i := 0; i < len(cfInfo); i++ {
			var c ConFlowCriteria
			c.ID = uint64(cfInfo[i].ID)
			c.Expression = cfInfo[i].Expression
			c.Repeat = cfInfo[i].Min

			cfMatched, err := ConversationFlowMatch(matchSgID, senGrpCriteria, &c, senGrpUUIDMapID, numOfLines)
			if err != nil {
				logger.Error.Printf("getting the conversation flow match failed. %s\n", err)
				return nil, err
			}
			if cfMatched {
				matchCFID[c.ID] = true
			}

			//stores the conversation flow level result
			senGrpIDs := conContainSenGrp[c.ID]
			credit := &ConversationFlowCredit{ID: c.ID, Valid: cfMatched}
			for _, senGrpID := range senGrpIDs {
				if v, ok := senGrpCreditMap[senGrpID]; ok {
					credit.SentenceGrps = append(credit.SentenceGrps, v)
				} else {
					logger.Error.Printf("sentence group id %d doesn't exist in the sentence group map\n", senGrpID)
					return nil, ErrRequestNotEqualGet
				}
			}
			cfCreditMap[c.ID] = credit
		}

		//--------------------------------------------------------------------------

		ruleGrpContainRule := levels[LevRuleGroup]
		ruleGrpIDs := make([]uint64, 0, len(ruleGrpContainRule))
		ruleIDs := make([]uint64, 0, len(ruleGrpContainRule))
		for rGrpID, ruleList := range ruleGrpContainRule {
			ruleGrpIDs = append(ruleGrpIDs, rGrpID)
			ruleIDs = append(ruleIDs, ruleList...)
		}

		ruleFilter := &model.ConversationRuleFilter{ID: ruleIDs, IsDeleted: -1, Severity: -1}
		_, rules, err := GetConversationRulesBy(ruleFilter)
		if err != nil {
			logger.Error.Printf("get the rules failed.%s\n", err)
			return nil, err
		}
		if len(rules) != len(ruleIDs) {
			logger.Error.Printf("request rules(%d) %v not equal to get %d\n", len(ruleIDs), ruleIDs, len(rules))
			return nil, ErrRequestNotEqualGet
		}

		ruleCriteria := make(map[uint64]*RuleCriteria)
		for _, v := range rules {
			c := &RuleCriteria{}
			c.ID = uint64(v.ID)
			c.Method = int(v.Method)
			c.Min = v.Min
			c.Score = v.Score
			for _, cfID := range v.Flows {
				c.CFIDs = append(c.CFIDs, uint64(cfID.ID))
			}
			ruleCriteria[c.ID] = c
		}

		matchRule, totalScore, err := RuleMatch(matchCFID, ruleCriteria)
		if err != nil {
			logger.Error.Printf("rule level match failed.%s\n", err)
			return nil, err
		}

		//stores the every result in every level
		var rgCredit RuleGrpCredit

		rgCredit.ID = uint64(ruleGroup.ID)
		rgCredit.Plus = totalScore
		rgCredit.Score = totalScore
		for _, ruleID := range ruleIDs {
			cfIDs := levels[LevRule][ruleID]
			credit := &RuleCredit{ID: ruleID}
			if v, ok := matchRule[ruleID]; ok {
				credit.Valid = v.Valid
				credit.Score = v.Score
			}

			for _, cfID := range cfIDs {
				if v, ok := cfCreditMap[cfID]; ok {
					credit.CFs = append(credit.CFs, v)
				} else {
					logger.Error.Printf("cannot get conversation flow %d credit in credit map\n", cfID)
					return nil, ErrRequestNotEqualGet
				}
			}
			rgCredit.Rules = append(rgCredit.Rules, credit)
		}
		rgCredit.Matched = tagMatchDat
		resp = append(resp, &rgCredit)
	}
	return resp, nil
}

//SimpleSentenceMatch gives the matched sentence id as key and index of segments that matched the sentence
//segs the string that wants to do check
//ids is the sentence id that wants to do the check
func SimpleSentenceMatch(segs []string, ids []uint64, enterprise string) (map[uint64][]int, error) {
	if len(segs) == 0 {
		return nil, ErrNeedSentence
	}
	if len(ids) == 0 {
		return nil, ErrNoArgument
	}

	//get the current model id by enterprise
	models, err := GetUsingModelByEnterprise(enterprise)
	if err != nil {
		logger.Error.Printf("get the model failed. %s\n", err.Error())
		return nil, err
	}
	numOfModels := len(models)
	if numOfModels == 0 {
		return nil, ErrNoModels
	} else if numOfModels > 1 {
		logger.Warn.Printf("More than 1 models is marked as using status with enterprise %s\n", enterprise)
		logger.Warn.Printf("Using the first one %d as prediction model", models[0].ID)
	}

	//get the relation table from Sentence to Tag
	levels, _, err := GetLevelsRel(LevSentence, LevTag, ids, true)
	if err != nil {
		logger.Error.Printf("get level relations failed. %s\n", err)
		return nil, err
	}

	if len(levels) != 1 {
		return nil, ErrLessRelation
	}

	//--------------------------------------------------------------------------
	numOfLines := len(segs)
	timeout := time.Duration(30 * time.Second)
	//do the checking, tag match
	tagMatchDat, err := TagMatch([]uint64{models[0].ID}, segs, timeout)
	if err != nil {
		return nil, fmt.Errorf("tag match failed, %v", err)
	}
	if len(tagMatchDat) != numOfLines {
		return nil, fmt.Errorf("get less tag match sentence %d with %d", len(tagMatchDat), numOfLines)
	}

	//--------------------------------------------------------------------------
	//do the sentence check
	// segMatchedTag struct []map[uint64]bool,
	// every slice index is a segment, which has a bunch of uint64(tag id),
	// use map for quick search later
	segMatchedTag := extractTagMatchedData(tagMatchDat)
	//do the checking, sentence match
	senMatchDat, err := SentencesMatchWithZeroBasedIndex(segMatchedTag, levels[0])
	if err != nil {
		logger.Warn.Printf("doing sentence  match failed.%s\n", err)
	}

	return senMatchDat, err
}
