package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func TestMain(m *testing.M) {
	LogInit(os.Stdout, os.Stdout, os.Stdout, os.Stdout)
	retCode := m.Run()
	os.Exit(retCode)
}

//TestConsulUpdateVal used mocked Http Server to verify the request is valid
//Instead of using a real consul server, it cut the time and dependency
func TestConsulUpdateVal(t *testing.T) {
	type kv struct {
		key string
		val interface{}
	}
	type testObject struct {
		A string `json:"a"`
		B bool   `json:"b"`
	}
	tables := map[string]kv{
		"字串": kv{"test", "hello"},
		//Know Issue: Golang will parse int as float64 in JSON.UnMarshal
		"數值": kv{"test", 1234.0},
		"JSON物件": kv{"test", testObject{
			A: "Hello", B: false,
		}},
	}
	for name, tt := range tables {
		t.Run(name, func(t *testing.T) {
			th := func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Fatalf("Expect HTTP Method be PUT, but got %v", r.Method)
					return
				}
				if uri := r.URL.RequestURI(); uri != "/"+tt.key {
					t.Fatalf("Expect URI to be /%s, but got %s", tt.key, uri)
				}
				data, err := ioutil.ReadAll(r.Body)
				defer r.Body.Close()
				if err != nil {
					t.Fatal(err)
					return
				}
				var jsonBody interface{}
				//This is needed because golang json type parsing problem.
				//struct became map lead to incomparable situcation
				switch tt.val.(type) {
				case testObject:
					var jsonBody testObject
					err = json.Unmarshal(data, &jsonBody)
					if err != nil {
						t.Fatal(err)
						return
					}
					if !reflect.DeepEqual(jsonBody, tt.val) {
						t.Fatalf("Expect test val be %T of %v, but got %T of %+v", tt.val, tt.val, jsonBody, jsonBody)
					}
				default:
					err = json.Unmarshal(data, &jsonBody)
					if err != nil {
						t.Fatal(err)
						return
					}
					if jsonBody != tt.val {
						t.Fatalf("Expect test val be %T of %v, but got %T of %+v", tt.val, tt.val, jsonBody, jsonBody)
					}
				}

			}
			ts := httptest.NewServer(http.HandlerFunc(th))
			defer ts.Close()
			DefaultConsulClient.Address = ts.URL
			ConsulUpdateVal(tt.key, tt.val)
		})
	}

}

func TestConsulUpdateTaskEngine(t *testing.T) {
	th := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+ConsulTEKey {
			t.Fatalf("expect URI to be /%v, but got %v", ConsulTEKey, r.URL.Path)
		}
	}

	ts := httptest.NewServer(http.HandlerFunc(th))
	defer ts.Close()
	DefaultConsulClient.Address = ts.URL
	ConsulUpdateTaskEngine("", true)
}

func TestConsulUpdateRobotChat(t *testing.T) {
	appid := "vipshop"
	th := func(w http.ResponseWriter, r *http.Request) {
		expectedURI := fmt.Sprintf("/"+ConsulRCKey, appid, appid)
		if r.URL.Path != expectedURI {
			t.Fatalf("expect URI to be /%v, but got %v", expectedURI, r.URL.Path)
		}
	}

	ts := httptest.NewServer(http.HandlerFunc(th))
	defer ts.Close()
	DefaultConsulClient.Address = ts.URL
	ConsulUpdateRobotChat(appid)
}
