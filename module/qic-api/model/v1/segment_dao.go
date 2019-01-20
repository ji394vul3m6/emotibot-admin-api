package model

import (
	"fmt"
	"strings"

	"emotibot.com/emotigo/pkg/logger"
)

type SegmentDao struct {
	db DBLike
}

func NewSegmentDao(db DBLike) *SegmentDao {
	return &SegmentDao{
		db: db,
	}
}

// RealSegment is the one to one mapping of the segment table schema.
// Do not confused with another vad segment which named Segment in this package too.
// It is the same thing but in different structure. Which used in different context.
//	Emotions is a virutal field for the relation with RealSegmentEmotion.
type RealSegment struct {
	ID         int64
	CallID     int64
	StartTime  float64
	EndTime    float64
	CreateTime int64
	Channel    int8
	Text       string
	Emotions   []RealSegmentEmotion
}

// RealSegmentEmotion is the one to one mapping of the SegEmotionScore table schema.
type RealSegmentEmotion struct {
	ID        int64
	SegmentID int64
	Typ       int8
	Score     float64
}

const (
	//ETypAngry is the value of angry(憤怒) emotion type for the RealSegmentEmotion
	ETypAngry = 1
)

// SegmentQuery is the AND query conditions for the segment table
type SegmentQuery struct {
	ID      []int64
	CallID  []int64
	Channel []int8
	page    *Pagination
}

func (s *SegmentQuery) whereSQL() (string, []interface{}) {
	builder := whereBuilder{
		ConcatLogic: andLogic,
		data:        []interface{}{},
		conditions:  []string{},
	}

	builder.In(fldSegmentID, int64ToWildCard(s.ID...))
	builder.In(fldSegmentCallID, int64ToWildCard(s.CallID...))
	builder.In(fldSegmentChannel, int8ToWildCard(s.Channel...))
	rawsql, data := builder.Parse()
	if len(data) > 0 {
		return " WHERE " + rawsql, data
	}
	return "", []interface{}{}
}

// Segments search the db by given query, and return a slice of found Segments
// notice: the segments will be sorted by start time, which is much more convenient for the users
func (s *SegmentDao) Segments(delegatee SqlLike, query SegmentQuery) ([]RealSegment, error) {
	if delegatee == nil {
		delegatee = s.db.Conn()
	}

	selectCols := []string{
		fldSegmentID, fldSegmentCallID, fldSegmentStartTime,
		fldSegmentEndTime, fldSegmentChannel, fldSegmentText,
	}
	wherepart, binddata := query.whereSQL()
	rawquery := "SELECT `" + strings.Join(selectCols, "`, `") + "` FROM `" + tblSegment + "` " + wherepart + " ORDER BY `" + fldSegmentStartTime + "` "
	if query.page != nil {
		rawquery += query.page.offsetSQL()
	}
	rows, err := delegatee.Query(rawquery, binddata...)
	if err != nil {
		logger.Error.Println("raw error sql: ", rawquery)
		logger.Error.Println("raw error data: ", binddata)
		return nil, fmt.Errorf("sql query error, %v", err)
	}
	defer rows.Close()
	var segments = []RealSegment{}
	for rows.Next() {
		var s RealSegment
		rows.Scan(&s.ID, &s.CallID, &s.StartTime, &s.EndTime, &s.Channel, &s.Text)
		segments = append(segments, s)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("sql scan error, %v", err)
	}
	return segments, nil
}

// NewSegments will insert the segments and its emotions into the segment & emotion database.
// Best use with a delegatee transcation to avoid data corruption.
func (s *SegmentDao) NewSegments(delegatee SqlLike, segments []RealSegment) ([]RealSegment, error) {
	if delegatee == nil {
		delegatee = s.db.Conn()
	}
	segCols := []string{
		fldSegmentCallID, fldSegmentStartTime, fldSegmentEndTime,
		fldSegmentCreateTime, fldSegmentChannel, fldSegmentText,
	}
	rawsql := "INSERT INTO `" + tblSegment + "` (`" + strings.Join(segCols, "`, `") + "`)  VALUE (?" + strings.Repeat(",?", len(segCols)-1) + ")"
	stmt, err := delegatee.Prepare(rawsql)
	if err != nil {
		logger.Error.Println("error raw sql: ", rawsql)
		return nil, fmt.Errorf("prepare segment insert statement failed, %v", err)
	}
	defer stmt.Close()
	emoCols := []string{
		fldSegEmoSegmentID, fldSegEmoType, fldSegEmoScore,
	}
	rawsql = "INSERT INTO `" + tblSegmentEmotion + "` (`" + strings.Join(emoCols, "`, `") + "`) VALUE (?" + strings.Repeat(",?", len(emoCols)-1) + ")"
	emotionStmt, err := delegatee.Prepare(rawsql)
	if err != nil {
		logger.Error.Println("error raw sql: ", rawsql)
		return nil, fmt.Errorf("prepare emotion insert statement failed, %v", err)
	}
	defer emotionStmt.Close()
	var hasIncreID = true
	for _, s := range segments {
		result, err := stmt.Exec(
			s.CallID, s.StartTime, s.EndTime,
			s.CreateTime, s.Channel, s.Text)
		if err != nil {
			return nil, fmt.Errorf("segment statement execute failed, %v", err)
		}
		s.ID, err = result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("segment can not get the inserted id, %v", err)
		}
		for _, e := range s.Emotions {
			e.SegmentID = s.ID
			eResult, err := emotionStmt.Exec(e.SegmentID, e.Typ, e.Score)
			if err != nil {
				return nil, fmt.Errorf("emotion statement execute failed, %v", err)
			}
			e.ID, err = eResult.LastInsertId()
			if err != nil {
				hasIncreID = false
			}
		}
	}
	if !hasIncreID {
		err = ErrAutoIDDisabled
	}
	return nil, err
}
