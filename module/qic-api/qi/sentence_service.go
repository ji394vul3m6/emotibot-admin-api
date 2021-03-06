package qi

import (
	"encoding/hex"
	"time"

	model "emotibot.com/emotigo/module/qic-api/model/v1"
	"emotibot.com/emotigo/pkg/logger"
	uuid "github.com/satori/go.uuid"
)

var (
	sentenceDao model.SentenceDao
)

//DataSentence data structures of sentence level
type DataSentence struct {
	ID         uint64     `json:"id"`
	CategoryID uint64     `json:"category_id,string"`
	UUID       string     `json:"sentence_id,omitempty"`
	Name       string     `json:"sentence_name,omitempty"`
	Tags       []*DataTag `json:"tags"`
}

//DataTag is data struct of tag level
type DataTag struct {
	UUID        string   `json:"tag_id,omitempty"`
	Name        string   `json:"tag_name,omitempty"`
	Type        string   `json:"tag_type,omitempty"`
	PosSentence []string `json:"pos_sentences,omitempty"`
	NegSentence []string `json:"neg_sentences,omitempty"`
}

//SrvSentence is used to as a structure to store the arguments which inputs to service in sentence
type SrvSentence struct {
	UUID       string
	Enterprise *string
	Name       *string
	Limit      int
	Page       int
	TagUUID    []string
}

//GetSentence gets one sentence depends on uuid
func GetSentence(uuid string, enterprise string) (*DataSentence, error) {
	var isDelete int8
	q := &model.SentenceQuery{UUID: []string{uuid}, IsDelete: &isDelete, Enterprise: &enterprise}
	data, err := getSentences(q)
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		return data[0], nil
	}
	return nil, nil
}

//GetSentenceList gets list of sentences queried by enterprise
//parameters:
//	isdelete, nil for no constraint, 0 for not delete, 1 for deleted
//	categortID, nil for no constraint, 0 for the sentences in unknown category, others for category id
//	sentenceName, empty for no constraint, anything will be a fuzzy search for sentence name(ex:app, will resulted as: apple, lapp)
func GetSentenceList(enterprise string, page int, limit int, isDelete *int8, categoryID *uint64, sentenceName string) (uint64, []*DataSentence, error) {
	//var isDelete int8
	q := &model.SentenceQuery{
		Enterprise: &enterprise,
		IsDelete:   isDelete,
		CategoryID: categoryID,
	}
	if sentenceName != "" {
		fuzzyName := model.EscapeLike(sentenceName)
		q.FuzzyName = fuzzyName
	}
	count, err := sentenceDao.CountSentences(nil, q)
	if err != nil {
		return 0, nil, err
	}

	q.Page = page
	q.Limit = limit

	if count > 0 {
		data, err := getSentences(q)
		return count, data, err
	}

	return count, []*DataSentence{}, nil
}

func getSentences(q *model.SentenceQuery) ([]*DataSentence, error) {
	sentences, err := sentenceDao.GetSentences(nil, q)
	if err != nil {
		return nil, err
	}
	numOfSens := len(sentences)
	if numOfSens == 0 {
		return nil, nil
	}
	data := make([]*DataSentence, 0, numOfSens)
	allTagIDs := make([]uint64, 0)
	for i := 0; i < numOfSens; i++ {
		d := &DataSentence{ID: sentences[i].ID, UUID: sentences[i].UUID, Name: sentences[i].Name, CategoryID: sentences[i].CategoryID}
		d.Tags = make([]*DataTag, 0)
		allTagIDs = append(allTagIDs, sentences[i].TagIDs...)
		data = append(data, d)
	}

	//get tags information
	query := model.TagQuery{ID: allTagIDs, Enterprise: q.Enterprise}

	tags, err := tagDao.Tags(nil, query)
	if err != nil {
		return nil, err
	}
	//transform tag data to map[tag_id] tag
	tagsIDMap := make(map[uint64]*model.Tag)
	for i := 0; i < len(tags); i++ {
		tagsIDMap[tags[i].ID] = &tags[i]
	}

	for i := 0; i < len(data); i++ {
		data[i].Tags = make([]*DataTag, 0)
		for _, tagID := range sentences[i].TagIDs {
			if tag, ok := tagsIDMap[tagID]; ok {
				dataTag := &DataTag{UUID: tag.UUID, Name: tag.Name, Type: tagTypeDict[tag.Typ]}
				/*
					dataTag.PosSentence = make([]string, 0)
					dataTag.NegSentence = make([]string, 0)

						if tag.PositiveSentence != "" {
							err = json.Unmarshal([]byte(tag.PositiveSentence), &dataTag.PosSentence)
							if err != nil {
								logger.Error.Printf("umarshal tag positive %s failed. %s", tag.PositiveSentence, err)
								return nil, err
							}
						}
						if tag.NegativeSentence != "" {
							err = json.Unmarshal([]byte(tag.NegativeSentence), &dataTag.NegSentence)
							if err != nil {
								logger.Error.Printf("umarshal tag negative %s failed. %s", tag.NegativeSentence, err)
								return nil, err
							}
						}
				*/

				data[i].Tags = append(data[i].Tags, dataTag)
			}
		}
	}

	return data, nil
}

//NewSentence inserts a new sentence
func NewSentence(enterprise string, category uint64, name string, tagUUID []string) (*DataSentence, error) {

	//query tags ID
	query := model.TagQuery{UUID: tagUUID, Enterprise: &enterprise}
	tags, err := tagDao.Tags(nil, query)
	if err != nil {
		return nil, err
	}

	numOfTags := len(tags)
	if numOfTags != len(tagUUID) {
		logger.Warn.Printf("user input tagUUID [%+v] not equals to tags [%+v] in db\n", tagUUID, tags)
	}

	tagsID := make([]uint64, numOfTags, numOfTags)
	for i := 0; i < numOfTags; i++ {
		tagsID[i] = tags[i].ID
	}

	//insert into the sentence
	now := time.Now().Unix()
	uuid, err := uuid.NewV4()
	if err != nil {
		logger.Error.Printf("generate uuid failed. %s\n", err)
		return nil, err
	}
	uuidStr := hex.EncodeToString(uuid[:])
	s := &model.Sentence{IsDelete: 0, Name: name, Enterprise: enterprise,
		CreateTime: now, UpdateTime: now, TagIDs: tagsID, UUID: uuidStr, CategoryID: category}
	tx, err := dbLike.Begin()
	if err != nil {
		logger.Error.Printf("create transaction failed. %s\n", err)
		return nil, err
	}
	defer tx.Rollback()
	newID, err := sentenceDao.InsertSentence(tx, s)
	if err != nil {
		return nil, err
	}
	s.ID = uint64(newID)
	err = sentenceDao.InsertSenTagRelation(tx, s)
	if err != nil {
		return nil, err
	}
	dbLike.Commit(tx)
	sentence := &DataSentence{UUID: uuidStr, Name: name}
	return sentence, nil
}

//UpdateSentence updates sentence, do soft delete and create new record
func UpdateSentence(sentenceUUID string, name string, enterprise string, tagUUID []string) (int64, error) {
	tx, err := dbLike.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	//soft delete the sentence
	var isDeleteInt int8
	q := &model.SentenceQuery{UUID: []string{sentenceUUID}, Enterprise: &enterprise, IsDelete: &isDeleteInt}
	sentences, err := sentenceDao.GetSentences(tx, q)
	if err != nil {
		logger.Error.Printf("query sentences failed. %s\n", err)
		return 0, err
	}

	if len(sentences) == 0 {
		return 0, nil
	}

	sentence := sentences[0]

	affected, err := sentenceDao.SoftDeleteSentence(tx, q)
	//means already deleted or non of rows is matched the condition
	if affected == 0 {
		return 0, nil
	}
	//query tags ID
	query := model.TagQuery{UUID: tagUUID, Enterprise: &enterprise}
	tags, err := tagDao.Tags(nil, query)
	if err != nil {
		logger.Error.Printf("Query tag failed. %s\n", err)
		return 0, err
	}
	numOfTags := len(tags)
	tagIDs := make([]uint64, numOfTags, numOfTags)
	for i := 0; i < numOfTags; i++ {
		tagIDs[i] = tags[i].ID
	}

	// get related sentence groups
	sgs, err := sentenceGroupDao.GetBySentenceID([]int64{int64(sentence.ID)}, tx)
	if err != nil {
		logger.Error.Printf("query setence groups failed. %s\n", err)
		return 0, err
	}

	//insert the sentence
	now := time.Now().Unix()
	s := &model.Sentence{Name: name, Enterprise: enterprise,
		UUID: sentenceUUID, CreateTime: now, UpdateTime: now,
		TagIDs: tagIDs, CategoryID: sentence.CategoryID}
	newID, err := sentenceDao.InsertSentence(tx, s)
	if err != nil {
		logger.Error.Printf("insert sentence failed. %s\n", err)
		return 0, err
	}
	s.ID = uint64(newID)
	err = sentenceDao.InsertSenTagRelation(tx, s)
	if err != nil {
		return 0, err
	}

	err = propagateUpdateFromSentenceGroup(sgs, []model.Sentence{*s}, tx)
	if err != nil {
		logger.Error.Printf("update sentence group relation failed. %s", err)
		return 0, err
	}

	err = dbLike.Commit(tx)
	if err != nil {
		return 0, err
	}

	return newID, nil
}

//SoftDeleteSentence sets the field, is_delete, in sentence to 1
func SoftDeleteSentence(sentenceUUID string, enterprise string) (int64, error) {
	tx, err := dbLike.Begin()
	if err != nil {
		return 0, nil
	}
	defer dbLike.ClearTransition(tx)

	var deleted int8
	q := &model.SentenceQuery{UUID: []string{sentenceUUID}, Enterprise: &enterprise, IsDelete: &deleted}
	sentences, err := sentenceDao.GetSentences(tx, q)
	if err != nil {
		logger.Error.Printf("query sentences failed. %s", err)
		return 0, err
	}

	if len(sentences) == 0 {
		return 0, nil
	}

	affected, err := sentenceDao.SoftDeleteSentence(tx, q)
	if err != nil {
		return affected, err
	}

	if affected == 0 {
		return 0, nil
	}

	sentence := sentences[0]

	sgs, err := sentenceGroupDao.GetBySentenceID([]int64{int64(sentence.ID)}, tx)
	if err != nil {
		logger.Error.Printf("query sentence group failed. %s", err)
		return 0, err
	}

	// remove sentence from sentence group
	for i := range sgs {
		sg := &sgs[i]
		logger.Info.Printf("sg: %+v\n", sg)
		if len(sg.Sentences) == 1 {
			sg.Sentences = []model.SimpleSentence{}
			continue
		}

		for j := range sg.Sentences {
			s := &sg.Sentences[j]
			if s.ID == sentence.ID {
				if j == len(sg.Sentences)-1 {
					sg.Sentences = sg.Sentences[:j]
				} else {
					sg.Sentences = append(sg.Sentences[:j], sg.Sentences[j+1:]...)
				}
			}
		}
	}

	err = propagateUpdateFromSentenceGroup(sgs, []model.Sentence{*sentence}, tx)
	if err != nil {
		return 0, err
	}

	err = dbLike.Commit(tx)
	return affected, err
}

func propagateUpdateFromSentenceGroup(sentenceGroups []model.SentenceGroup, sentences []model.Sentence, sqlLike model.SqlLike) (err error) {
	logger.Info.Printf("sentenceGroups: %+v", sentenceGroups)
	logger.Info.Printf("sentences: %+v", sentences)
	if len(sentenceGroups) == 0 || len(sentences) == 0 {
		return
	}

	// update sentence id in sentence groups
	sentenceMap := map[string]int64{}
	for _, sentence := range sentences {
		sentenceMap[sentence.UUID] = int64(sentence.ID)
	}

	sgUUID := []string{}
	sgID := []int64{}
	activeSentenceGroups := []model.SentenceGroup{} // only update non-deleted sentence groups
	for i := range sentenceGroups {
		group := &sentenceGroups[i]
		if group.Deleted == 1 {
			// ignore deleted groups
			continue
		}

		for j := range group.Sentences {
			sentence := &group.Sentences[j]
			if sentenceID, ok := sentenceMap[sentence.UUID]; ok {
				sentence.ID = uint64(sentenceID)
			}
		}
		sgUUID = append(sgUUID, group.UUID)
		sgID = append(sgID, group.ID)
		activeSentenceGroups = append(activeSentenceGroups, *group)
	}

	// get related flows of the sentence groups
	flows, err := conversationFlowDao.GetBySentenceGroupID(sgID, sqlLike)
	if err != nil {
		return
	}

	// delete old sentence groups
	err = sentenceGroupDao.DeleteMany(sgUUID, sqlLike)
	if err != nil {
		return
	}

	// create new sentence groups
	err = sentenceGroupDao.CreateMany(activeSentenceGroups, sqlLike)
	if err != nil {
		return
	}

	return propagateUpdateFromFlow(flows, activeSentenceGroups, sqlLike)
}

//CheckSentenceAuth checks whether these uuid belongs to this enterprise
func CheckSentenceAuth(sentenceUUID []string, enterprise string) (bool, error) {

	numOfSen := len(sentenceUUID)

	var isDelete int8
	q := &model.SentenceQuery{UUID: sentenceUUID, Enterprise: &enterprise, IsDelete: &isDelete}
	count, err := sentenceDao.CountSentences(nil, q)
	if err != nil {
		return false, err
	}
	if count < uint64(numOfSen) {
		return false, nil
	}
	return true, nil
}

//MoveCategories moves the sentence to the assigned category
func MoveCategories(sentenceUUID []string, enterprise string, category uint64) (int64, error) {
	q := &model.SentenceQuery{UUID: sentenceUUID, Enterprise: &enterprise}
	return sentenceDao.MoveCategories(nil, q, category)
}
