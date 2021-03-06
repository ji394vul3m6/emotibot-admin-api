package v1

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	dataV1 "emotibot.com/emotigo/module/admin-api/ELKStats/data/v1"
	"emotibot.com/emotigo/module/admin-api/util/elasticsearch"
)

func TestUpdateRecords(t *testing.T) {
	// isIntegrate := flag.Bool("integrate", false, "run integration test")
	// flag.Parse()
	// if !*isIntegrate {
	// 	t.Skip("integration only run with --integrate flag")
	// }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if path := r.URL.String(); path != "emotibot-records-csbot-*" {
		// 	t.Fatalf("expect request url to be emotibot-records-csbot-* but got %s", path)
		// }
		data, _ := ioutil.ReadAll(r.Body)
		fmt.Println(string(data))
		// expectQuery := `{"query":{"bool":{"filter":{"terms":{"unique_id":["20180823183657462364352"]}}}},"script":{"params":{"mark":true},"source":"ctx._source.isMarked = params.mark"}}`
		// if string(data) != expectQuery {
		// 	t.Fatalf("expect query to be %s but got %s", expectQuery, data)
		// }
	}))
	addr, _ := url.Parse(server.URL)
	baseAuthUsername := "username"
	baseAuthPassword := "password"
	err := elasticsearch.Setup(addr.Hostname(), addr.Port(), baseAuthUsername, baseAuthPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	VisitRecordsQuery(dataV1.RecordQuery{
		AppID:   "csbot",
		Records: []interface{}{"20180823183657462364352"},
		Limit:   3})
	return
	cmd := UpdateRecordMark(true)
	err = UpdateRecords(dataV1.RecordQuery{
		AppID:   "csbot",
		Records: []interface{}{"20180823183657462364352"}}, cmd)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewBoolQueryWithRecordQuery(t *testing.T) {
	type testCase struct {
		input string
		query string
	}
	var testTable = map[string]testCase{
		"keyword": {
			input: `{"keyword": "test"}`,
			query: `{"bool":{"filter":{"bool":{"should":[{"match":{"user_q":{"query":"test"}}},{"nested":{"path":"answer","query":{"match":{"answer.value":{"query":"test"}}}}}]}}}}`,
		},
		"searchByTime": {
			input: `{"start_time":1530439260,"end_time":1535364060}`,
			query: `{"bool":{"filter":{"range":{"log_time":{"format":"yyyy-MM-dd HH:mm:ss","from":"2018-07-01 10:01:00","include_lower":true,"include_upper":true,"time_zone":"+00:00","to":"2018-08-27 10:01:00"}}}}}`,
		},
	}
	for name, tc := range testTable {
		t.Run(name, func(tt *testing.T) {
			var q dataV1.RecordQuery
			json.Unmarshal([]byte(tc.input), &q)
			bq := newBoolQueryWithRecordQuery(&q)
			src, _ := bq.Source()
			query, _ := json.Marshal(src)
			if string(query) != tc.query {
				tt.Fatalf("expect query to be %s, but got %s", tc.query, query)
			}

		})
	}
}
