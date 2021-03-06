package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"emotibot.com/emotigo/pkg/api/v1/taskengine"
	"github.com/siongui/gojianfan"
)

type v2TextResponse struct {
	Text string `json:"text"`
}

func voiceToTextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintln(w, "Only support POST method.")
		return
	}
	var (
		data []byte
		err  error
	)
	aw, ok := w.(*AudioResponseWriter)
	if ok && aw.AudioFile != nil {
		data = aw.AudioFile
	} else {
		data, err = ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	current := time.Now().UnixNano()
	if len(data) == 0 {
		http.Error(w, "audil file is empty", http.StatusBadRequest)
		return
	}
	sentence, err := asrClient.Recognize(bytes.NewBuffer(data))
	asrTotalNanoTime := time.Now().UnixNano() - current
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
		log.Println(err)
		return
	}
	var resp v2TextResponse
	log.Printf("v2Text: %s\n", sentence)
	aw.Logs = []string{sentence, strconv.FormatInt(asrTotalNanoTime, 10)}
	resp.Text = sentence

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	data, err = json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	w.Write(data)
}

func voiceToTaskHandler(w http.ResponseWriter, r *http.Request) {
	timer := time.Now().UnixNano()
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintln(w, "Only support POST method.")
		return
	}
	var err error
	var userID = r.Header.Get("X-UserID")
	if userID == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "UserID is empty")
		return
	}
	var rawBody []byte
	aw, ok := w.(*AudioResponseWriter)
	if !ok {
		log.Println("no middleware have read the body yet, read it.")
		rawBody, err = ioutil.ReadAll(r.Body)
		if err != nil {
			message := "read body failed, " + err.Error()
			CustomError(aw, message)
			return
		}
	} else {
		rawBody = aw.AudioFile
	}

	current := time.Now().UnixNano()
	sentence, err := asrClient.Recognize(bytes.NewBuffer(rawBody))
	if err != nil {
		CustomError(aw, "asr api error:"+err.Error())
		return
	}
	asrTotalNanoTime := time.Now().UnixNano() - current

	sentence, err = csClient.Simplify(sentence)
	if err != nil {
		CustomError(aw, "Cu Service api error:"+err.Error())
		return
	}
	current = time.Now().UnixNano()
	resp, err := teClient.ET(userID, sentence)
	if err != nil {
		CustomError(aw, "task engine error:"+err.Error())
		return
	}
	teTotalNanoTime := time.Now().UnixNano() - current
	teResp, err := taskengine.ParseETResponse(resp)
	if err != nil {
		CustomError(aw, "task engine response err:"+err.Error())
		return
	}
	var out = Response{
		Text:       gojianfan.S2T(teResp.Text),
		ScenarioID: teResp.ScenarioID,
		Flag:       teResp.Flag,
		AsrText:    sentence,
	}
	outJSONData, err := json.Marshal(out)
	if err != nil {
		CustomError(aw, "response json err: "+err.Error())
		return
	}

	log.Printf("v2Task: %s, te result: %v\n", sentence, out)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(outJSONData)
	totalNanoTime := time.Now().UnixNano() - timer
	aw.Logs = []string{string(outJSONData), strconv.FormatInt(totalNanoTime, 10), strconv.FormatInt(asrTotalNanoTime, 10), strconv.FormatInt(teTotalNanoTime, 10)}
}

type Response struct {
	Text       string `json:"TTSText"`
	ScenarioID string `json:"HitScenarioID"`
	Flag       int    `json:"FinalFlag"`
	AsrText    string `json:"asr_text"`
}

//Custom
func CustomError(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusInternalServerError)
	log.Println(message)
	fmt.Fprintln(w, message)
}
