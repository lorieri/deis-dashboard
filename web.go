package main

import (
	"fmt"
	"net/http"
	"text/template"
        "gopkg.in/redis.v2"
	"strconv"
	"encoding/json"
	"os"
	"github.com/coreos/go-etcd/etcd"
	"strings"
)

var redisServer string
var etcdServers []string

func main() {
	getetcdServers := getopt("ETCD_HOSTS", "")
	if getetcdServers == "" {
		panic("Please set ETCD_HOSTS environment, comma separated http:// hosts with port")
	}
	etcdServers = strings.Split(getetcdServers, ",")
	client := etcd.NewClient(etcdServers)
	fmt.Println(client)
	fmt.Println("Etcd Servers:")
	fmt.Println(etcdServers)
	setRedis()
	http.HandleFunc("/var.json", vars)
	http.HandleFunc("/apps.json", varappsname)
	http.HandleFunc("/", dashboard)
	http.HandleFunc("/apps/",apps)
	http.HandleFunc("/apps/var.json", varsapps)
	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
	    http.ServeFile(w, r, r.URL.Path[1:])
	})
	port := "6969"
	fmt.Printf("listening on %v...\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}

// copied from github.com/deis
func getopt(name, dfault string) string {
        value := os.Getenv(name)
        if value == "" {
                value = dfault
        }
        return value
}


// http://blog.gopheracademy.com/advent-2013/day-06-service-discovery-with-etcd/
func updateRedis(){
        client := etcd.NewClient(etcdServers)
        watchChan := make(chan *etcd.Response)
        go client.Watch("/deis/logs/host", 0, false, watchChan, nil)
        resp := <-watchChan
        redisServer = resp.Node.Value
        updateRedis()
}

func setRedis(){
        redisServer = getopt("REDIS_SERVER", "")
	fmt.Println("Set redisServer as "+redisServer)
	if redisServer == "" {
	        client := etcd.NewClient(etcdServers)
		resp, err := client.Get("/deis-dashboard/redis", false, false)
		if err != nil {
			panic(err)
		}
		fmt.Println("Set Redis Server as "+resp.Node.Value)
	        redisServer = resp.Node.Value
		go updateRedis()
	}
}


type Page struct {
	Title string
	Json  string //[]byte
	App   string
}

type AppPage struct {
	Title           string
	App	         string
	RequestsSecond   float64
	TotalRequests    float64
	TotalSuccess     float64
	TotalErrors      float64
	BytesPerSecond   float64
	PercentErrors    string
	PercentSuccess   string
	Requests         map[string]float64
	StatusByUnits    map[string]float64
        Referers         map[string]float64
        ErrorsByReferers map[string]float64
	ErrorsByRequests map[string]float64
	RequestsByStatus map[string]float64
	RemoteByStatus   map[string]float64
	RemoteBytesSent  map[string]float64
}

type Var struct {
	LegendData    string
	XaxisData     string
	ErrorsData    string
	SuccessData   string
	TotalData     string
	TotalRequests string
	PieData       string
	PieDataBytes  string
	LastLog       string
}

func apps(w http.ResponseWriter, r *http.Request) {
        title := "Apps"
	app := r.URL.Path[len("/apps/"):]
        t, _ := template.ParseFiles("app.html")

        client := redis.NewClient(&redis.Options{Network: "tcp", Addr: redisServer})

        result, _ := client.ZRevRangeWithScores("union_z_top_app_upstream_status_"+app, "0" , "10").Result()
	upstreamstatus := make(map[string]float64)
	for _,v := range result {
		upstreamstatus[v.Member] = v.Score
	}

	totalrequests_str, _ := client.Get("union_k_total_app_requests_"+app).Result()
	totalrequests, _     := strconv.ParseFloat(totalrequests_str,64)
	requestsseconds      := totalrequests/10

	bytessent_str,_      := client.Get("union_k_total_app_bytes_sent_"+app).Result()
	bytessent,_	     := strconv.ParseFloat(bytessent_str,64)
	bytespersecond       := bytessent/10

	totalerrors_str, _   := client.Get("union_k_total_app_errors_"+app).Result()
	totalerrors, _       := strconv.ParseFloat(totalerrors_str,64)

	totalsuccess         := totalrequests-totalerrors

        result, _ = client.ZRevRangeWithScores("union_z_top_app_request_"+app, "0" , "10").Result()
	toprequests := make(map[string]float64)
	for _,v := range result {
		toprequests[v.Member] = v.Score
	}

	result, _ = client.ZRevRangeWithScores("union_z_top_app_error_request_"+app, "0" , "10").Result()
	toperrors := make(map[string]float64)
	for _,v := range result {
		toperrors[v.Member] = v.Score
	}

	result, _ = client.ZRevRangeWithScores("union_z_top_app_error_referer_"+app, "0" , "10").Result()
	toperrorsreferer := make(map[string]float64)
	for _,v := range result {
		toperrorsreferer[v.Member] = v.Score
	}

	result, _ = client.ZRevRangeWithScores("union_z_top_app_referer_"+app, "0" , "10").Result()
	topreferers := make(map[string]float64)
	for _,v := range result {
		topreferers[v.Member] = v.Score
	}

	result, _ = client.ZRevRangeWithScores("union_z_top_app_status_"+app, "0" , "10").Result()
	topstatus := make(map[string]float64)
	for _,v := range result {
		topstatus[v.Member] = v.Score
	}

	result, _ = client.ZRevRangeWithScores("union_z_top_remote_addr_bytes_sent_"+app, "0" , "10").Result()
	remotebytessent := make(map[string]float64)
	for _,v := range result {
		remotebytessent[v.Member] = v.Score
	}

	result, _ = client.ZRevRangeWithScores("union_z_top_remote_addr_status_"+app, "0" , "10").Result()
	remotebystatus := make(map[string]float64)
	for _,v := range result {
		remotebystatus[v.Member] = v.Score
	}

        p := &AppPage{
		Title: title,
		App: app,
		RequestsSecond:   requestsseconds,
		TotalRequests:    totalrequests,
		TotalSuccess:     totalsuccess,
		TotalErrors:      totalerrors,
		BytesPerSecond:   bytespersecond,
		Requests:         toprequests,
		PercentErrors:    strconv.FormatFloat(totalerrors/totalrequests*100, 'f', 1, 64),
		PercentSuccess:   strconv.FormatFloat(totalsuccess/totalrequests*100, 'f', 1, 64),
		StatusByUnits:    upstreamstatus,
		Referers:         topreferers, // wikipedia: HTTP referer (originally a misspelling of referrer)
		ErrorsByReferers: toperrorsreferer,
		ErrorsByRequests: toperrors,
		RequestsByStatus: topstatus,
		RemoteByStatus:   remotebystatus,
		RemoteBytesSent:  remotebytessent,
	}

        t.Execute(w, p)
}

func varappsname(w http.ResponseWriter, r *http.Request){
	client   := redis.NewClient(&redis.Options{Network: "tcp", Addr: redisServer})
        apps, _  := client.ZRangeWithScores("union_z_top_apps", 0 , -1).Result()

	var appsjson string
        for k,_ := range apps {
                appsjson      = appsjson+"{name: '"+apps[k].Member+"'},"
        }

	if len(appsjson) == 0 {
		appsjson = "{name:'none'}"
	}

	fmt.Fprintf(w, "["+appsjson+"]")
}

func varsapps(w http.ResponseWriter, r *http.Request){
}

func vars(w http.ResponseWriter, r *http.Request){
	client           := redis.NewClient(&redis.Options{Network: "tcp", Addr: redisServer})
	apps, _          := client.ZRangeWithScores("union_z_top_apps", 0 , -1).Result()
	appbytes, _      := client.ZRangeWithScores("union_z_top_apps_bytes_sent", 0, -1).Result()
	lastlog_str,_        := client.Get("union_s_last_log_time").Result()
	totaldata_str,_      := client.Get("union_k_total_bytes").Result()
        totalrequests_str, _ := client.Get("union_k_total_requests").Result()


	dataapps  := make([]string, 0)
	dataerror := make([]int, 0)
	datasucc  := make([]int, 0)
	datapie_str := ""
	datapiebytes_str := ""

	for k,_ := range apps {
		appname      := apps[k].Member
		apptotal_str := strconv.FormatFloat(apps[k].Score, 'f', 0, 64)
		apptotal     := apps[k].Score
		apperr_str,_ := client.Get("union_k_total_app_errors_"+appname).Result()
		apperr,_     := strconv.Atoi(apperr_str)

		appsuc       := int(apptotal)-int(apperr)
		if appsuc < 0 {
			appsuc = 0
		}
		// appreqs := apptotal/10

		dataapps    = append(dataapps, appname)
		dataerror   = append(dataerror, apperr)
		datasucc    = append(datasucc, appsuc)
		datapie_str = datapie_str+"{value: "+apptotal_str+", name: '"+appname+"'}," // ...
	}
	dataapps_legend := append(dataapps, "Success","Errors")

	for k,_ := range appbytes {
		appnamebytes      := appbytes[k].Member
		apptotalbytes_str := strconv.FormatFloat(appbytes[k].Score, 'f', 0, 64)
		datapiebytes_str   = datapiebytes_str+"{value: "+apptotalbytes_str+", name: '"+appnamebytes+"'},"
	}

	legend_data,_  := json.Marshal(dataapps_legend) //"['go','java','ruby','Success','Errors']"
	legend_data_str := string(legend_data)


	//&{LegendData:["Success","Errors"] XaxisData:[] ErrorsData:[] SuccessData:[] TotalData: TotalRequests: PieData:[] PieDataBytes:[] LastLog:}

	var xaxis_data_str string
	if len(dataapps) == 0 {
		xaxis_data_str = "[0]"
	}else{
		xaxis_data,_      := json.Marshal(dataapps)        //"['go','java','ruby']"
		xaxis_data_str     = string(xaxis_data)
	}

	var errors_data_str string
	if len(dataerror) == 0 {
		errors_data_str    = "[0]"
	}else{
		errors_data,_     := json.Marshal(dataerror)       //"[100,50,30]"
		errors_data_str    = string(errors_data)
	}

	var success_data_str string
	if len(datasucc) == 0 {
		success_data_str   = "[0]"
	}else{
		success_data,_    := json.Marshal(datasucc)        //"[100,50,30]"
		success_data_str   = string(success_data)
	}

	if totaldata_str == "" {
		totaldata_str      = "0"
	}

	if totalrequests_str == "" {
		totalrequests_str  = "0"
	}

	var pie_data_str string
	if  datapie_str == "" {
		pie_data_str       = "[{value:0, name:'none'}]"
	}else{
		pie_data_str       = "["+datapie_str+"]" //"[{value:1048, name:'go'},{value:251, name:'java'},{value:600, name:'ruby'},]"
	}

	var pie_data_bytes_str string
	if datapiebytes_str == "" {
		pie_data_bytes_str = "[{value:0, name:'none'}]"
	}else{
		pie_data_bytes_str = "["+datapiebytes_str+"]"
	}

	if lastlog_str == "" {
		lastlog_str        = "-"
	}


	p := &Var {
		LegendData:    legend_data_str,
		XaxisData:     xaxis_data_str,
		ErrorsData:    errors_data_str,
		SuccessData:   success_data_str,
		PieData:       pie_data_str,
		PieDataBytes:  pie_data_bytes_str,
		LastLog:       lastlog_str,
		TotalData:     totaldata_str,
		TotalRequests: totalrequests_str,
	}
	t, _ := template.ParseFiles("data.json")
	t.Execute(w, p)
}

func dashboard(w http.ResponseWriter, r *http.Request) {
	title := "Top Requests"
	t, _ := template.ParseFiles("dash.html")

	p := &Page{ Title: title }
	t.Execute(w, p)
}
