package qi

import (
	"testing"

	model "emotibot.com/emotigo/module/qic-api/model/v1"
	"emotibot.com/emotigo/module/qic-api/util/test"
)

type mockSQLRelationDao struct {
}

func (m *mockSQLRelationDao) GetLevelRelationID(sql model.SqlLike, from int, to int, id []uint64, ignoreNULL bool) ([]map[uint64][]uint64, [][]uint64, error) {
	return nil, nil, nil
}

func mockRelationDoa() {
	relationDao = &mockSQLRelationDao{}
}
func TestGetLevelsRel(t *testing.T) {
	mockRelationDoa()
	mockDBLike := &test.MockDBLike{}
	dbLike = mockDBLike
	var from, to Levels

	from = LevRuleGroup
	to = LevRuleGroup
	id := []uint64{1, 3, 5}
	_, _, err := GetLevelsRel(from, to, id, true)
	if err == nil {
		t.Error("expecting get error, but get no error\n")
	}

	from = LevSentence
	to = LevRuleGroup

	_, _, err = GetLevelsRel(from, to, id, true)
	if err == nil {
		t.Error("expecting get error, but get no error\n")
	}

	from = LevRuleGroup
	to = LevSentence
	_, _, err = GetLevelsRel(from, to, id, true)
	if err != nil {
		t.Errorf("expecting no error, but get error %s\n", err)
	}

}
