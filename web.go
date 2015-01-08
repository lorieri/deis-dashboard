package main

import (
	"fmt"
	"net/http"
	"text/template"
        "gopkg.in/redis.v2"
	"strconv"
	"encoding/json"
)

func main() {
	http.HandleFunc("/var.json", vars)
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
	PercentErrors    string
	PercentSuccess   string
	Requests         map[string]float64
	StatusByUnits    map[string]float64
        Referers         map[string]float64
        ErrorsByReferers map[string]float64
	ErrorsByRequests map[string]float64
	RequestsByStatus map[string]float64
}

type Var struct {
	LegendData  string
	XaxisData   string
	ErrorsData  string
	SuccessData string
	//TotalData   string
	PieData     string
}

func apps(w http.ResponseWriter, r *http.Request) {
        title := "Apps"
	app := r.URL.Path[len("/apps/"):]
        t, _ := template.ParseFiles("app.html")

        client := redis.NewClient(&redis.Options{Network: "tcp", Addr: "127.0.0.1:6379"})

        result, _ := client.ZRevRangeWithScores("union_z_top_app_upstream_status_"+app, "0" , "10").Result()
	upstreamstatus := make(map[string]float64)
	for _,v := range result {
		upstreamstatus[v.Member] = v.Score
	}

	totalrequests_str, _ := client.Get("union_k_total_app_requests_"+app).Result()
	totalrequests, _     := strconv.ParseFloat(totalrequests_str,32)
	requestsseconds      := totalrequests/10

	totalerrors_str, _   := client.Get("union_k_total_app_errors_"+app).Result()
	totalerrors, _       := strconv.ParseFloat(totalerrors_str,32)

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

        p := &AppPage{
		Title: title,
		App: app,
		RequestsSecond:   requestsseconds,
		TotalRequests:    totalrequests,
		TotalSuccess:     totalsuccess,
		TotalErrors:      totalerrors,
		Requests:         toprequests,
		PercentErrors:    strconv.FormatFloat(totalerrors/totalrequests*100, 'f', 1, 64),
		PercentSuccess:   strconv.FormatFloat(totalsuccess/totalrequests*100, 'f', 1, 64),
		StatusByUnits:    upstreamstatus,
		Referers:         topreferers, // wikipedia: HTTP referer (originally a misspelling of referrer)
		ErrorsByReferers: toperrorsreferer,
		ErrorsByRequests: toperrors,
		RequestsByStatus: topstatus,
	}
        t.Execute(w, p)
}

func varsapps(w http.ResponseWriter, r *http.Request){
}

func vars(w http.ResponseWriter, r *http.Request){
	client := redis.NewClient(&redis.Options{Network: "tcp", Addr: "127.0.0.1:6379"})
	apps, _ := client.ZRangeWithScores("union_z_top_apps", 0 , -1).Result()

	dataapps  := make([]string, 0)
	dataerror := make([]int, 0)
	datasucc  := make([]int, 0)
	datapie_str := ""

	for k,_ := range apps {
		appname := apps[k].Member
		apptotal_str := strconv.FormatFloat(apps[k].Score, 'f', 0, 64)
		apptotal := apps[k].Score
		apperr_str,_ := client.Get("union_k_total_app_errors_"+appname).Result()
		apperr,_ := strconv.Atoi(apperr_str)

		appsuc := int(apptotal)-int(apperr)
		if appsuc < 0 {
			appsuc = 0
		}
		// appreqs := apptotal/10

		dataapps  = append(dataapps, appname)
		dataerror = append(dataerror, apperr)
		datasucc  = append(datasucc, appsuc)
		datapie_str = datapie_str+"{value: "+apptotal_str+", name: '"+appname+"'}," // ...
	}
	dataapps_legend := append(dataapps, "Success","Errors")

	legend_data,_  := json.Marshal(dataapps_legend) //"['go','java','ruby','Success','Errors']"
	xaxis_data,_   := json.Marshal(dataapps)        //"['go','java','ruby']"
	errors_data,_  := json.Marshal(dataerror)       //"[100,50,30]"
	success_data,_ := json.Marshal(datasucc)        //"[100,50,30]"
	pie_data       := "["+datapie_str+"]"           //"[{value:1048, name:'go'},{value:251, name:'java'},{value:600, name:'ruby'},]"

	p := &Var {
		LegendData:  string(legend_data),
		XaxisData:   string(xaxis_data),
		ErrorsData:  string(errors_data),
		SuccessData: string(success_data),
		PieData:     pie_data,
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
