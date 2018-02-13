package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"

	"strings"

	"emotibot.com/emotigo/module/proxy/traffic"
)

var AddTrafficChan chan string
var ReadDestChan chan *trafficStats.RouteMap
var AppidChan chan *trafficStats.AppidIP
var trafficManager *trafficStats.TrafficManager

var k8sRedirectList = make(map[string]bool, 0)

func GoProxy(w http.ResponseWriter, r *http.Request) {

	buf, _ := ioutil.ReadAll(r.Body)

	rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
	rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf))

	r.Body = rdr1
	r.ParseMultipartForm(0)

	appid := ""
	userid := ""
	openapiCmd := ""

	if r.Method == "GET" || r.Method == "POST" {
		appid = r.FormValue("appid")
		openapiCmd = r.FormValue("cmd")
		// userid: OpenAPI
		userid = r.FormValue("userid")
		// UserID: /api/APP/chat.php  # FreemeOS
		userid += r.FormValue("UserID")
		// All other IDs:
		// phthon OpenID WeChatID wechatid user_id
		userid += r.FormValue("phthon")
		userid += r.FormValue("OpenID")
		userid += r.FormValue("WeChatID")
		userid += r.FormValue("wechatid")
		userid += r.FormValue("user_id")
	} else {
		// FIXME: Should we drop non GET/POST requests?
		log.Printf("Warning: Unknown request type. %s %s %s", r.Host, r.Method, string(buf))
		http.Error(w, "Method Error", http.StatusBadGateway)
		return
	}

	ipPort := strings.Split(r.RemoteAddr, ":")
	if len(ipPort) == 2 {
		do, ok := trafficStats.MonitorAppid[appid]
		if do && ok {
			appidIP := new(trafficStats.AppidIP)
			appidIP.Appid = appid
			appidIP.IP = ipPort[0]
			appidIP.Userid = userid
			AppidChan <- appidIP
		}

	} else {
		log.Printf("Warning: ip:port not fit. %s\n", r.RemoteAddr)
	}

	if k8sRedirectList[userid] {
		r.Header.Set("X-Lb-K8s", "k8suser")
	} else if trafficManager.CheckOverFlowed(userid) {
		userid = userid + strconv.Itoa(rand.Intn(1000))
	}

	r.Header.Set("X-Lb-Uid", userid)
	r.Header.Set("X-Openapi-Appid", appid)
	r.Header.Set("X-Openapi-Cmd", openapiCmd)

	r.Body = rdr2
	proxy := httputil.NewSingleHostReverseProxy(&trafficManager.Route)
	proxy.ServeHTTP(w, r)

}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func checkerr(err error, who string) {
	if err != nil {
		log.Fatalf("No %s env variable, %v\n", who, err)
	}
}

func main() {

	duration, err := strconv.Atoi(os.Getenv("DURATION"))
	checkerr(err, "DURATION")
	maxLimit, err := strconv.Atoi(os.Getenv("MAXREQUESTS"))
	checkerr(err, "MAXREQUESTS")
	banPeriod, err := strconv.Atoi(os.Getenv("BANPERIOD"))
	checkerr(err, "BANPERIOD")
	logPeriod, err := strconv.Atoi(os.Getenv("LOGPERIOD"))
	checkerr(err, "LOGPERIOD")
	statsdHost := os.Getenv("STATSDHOST")
	if statsdHost == "" {
		log.Fatal("No STATSDHOST")
	}
	statsdPort := os.Getenv("STATSDPORT")
	if statsdPort == "" {
		log.Fatal("No STATSDPORT")
	}

	log.Printf("Setting max %d request in %d seconds, banned period %d, log period:%d\n", maxLimit, duration, banPeriod, logPeriod)
	f, err := os.Open("./k8slist")
	if err != nil {
		log.Fatalf("read ./k8slist failed, %v", err)
	}
	k8sRedirectList, err := ReadList(f)
	if err != nil {
		log.Fatalf("ReadList failed, %v", err)
	}
	log.Printf("K8S List readed:\n%+v\n", k8sRedirectList)
	//make the channel
	AddTrafficChan = make(chan string)
	ReadDestChan = make(chan *trafficStats.RouteMap)
	AppidChan = make(chan *trafficStats.AppidIP, 1024)
	u, _ := url.Parse("http://172.17.0.1:9001")
	trafficManager = trafficStats.NewTrafficManager(duration, int64(maxLimit), int64(banPeriod), *u)
	go trafficStats.AppidCounter(logPeriod, statsdHost+":"+statsdPort)
	http.HandleFunc("/", GoProxy)
	http.HandleFunc("/_health_check", HealthCheck)
	log.Fatal(http.ListenAndServe(":9000", nil))

}

func ReadList(r io.Reader) (map[string]bool, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read reader failed, %v", err)
	}
	lists := strings.Split(string(data), "\n")
	list := make(map[string]bool, len(lists))
	for _, item := range lists {
		//Skip # comment
		if strings.HasPrefix(item, "#") {
			continue
		}
		list[item] = true
	}
	return list, nil
}
