package lib

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/inconshreveable/log15"
	model "github.com/netsec-ethz/scion-apps/webapp/models"
	. "github.com/netsec-ethz/scion-apps/webapp/util"
)

// results data extraction regex
var reRunTimeS = `(packet loss, time )(\d*\.?\d*)(s)`
var reRunTimeMs = `(packet loss, time )(\d*\.?\d*)(ms)`
var reRunTimeUs = `(packet loss, time )(\d*\.?\d*)(µs)`
var reRespTimeS = `(scmp_seq=0 time=)(\d*\.?\d*)(s)`
var reRespTimeMs = `(scmp_seq=0 time=)(\d*\.?\d*)(ms)`
var reRespTimeUs = `(scmp_seq=0 time=)(\d*)(µs)`
var rePktLoss = `(\d+)(% packet loss,)`

// ExtractEchoRespData will parse cmd line output from scmp echo for adding EchoItem fields.
func ExtractEchoRespData(resp string, d *model.EchoItem) {
	// store current epoch in ms
	d.Inserted = time.Now().UnixNano() / 1e6

	log.Info("resp response", "content", resp)

	var data = make(map[string]string)
	var path, err string
	var match bool
	pathNext := false
	r := strings.Split(resp, "\n")
	for i := range r {
		// match response time in unit s
		match, _ = regexp.MatchString(reRespTimeS, r[i])
		if match {
			re := regexp.MustCompile(reRespTimeS)
			tStr := re.FindStringSubmatch(r[i])[2]
			t, _ := strconv.ParseFloat(tStr, 32)
			data["response_time"] = fmt.Sprintf("%f", (t * 1000))
		}
		// match response time in unit ms
		match, _ = regexp.MatchString(reRespTimeMs, r[i])
		if match {
			re := regexp.MustCompile(reRespTimeMs)
			tStr := re.FindStringSubmatch(r[i])[2]
			t, _ := strconv.ParseFloat(tStr, 32)
			data["response_time"] = fmt.Sprintf("%f", t)
		}
		// match response time in unit μs
		match, _ = regexp.MatchString(reRespTimeUs, r[i])
		if match {
			re := regexp.MustCompile(reRespTimeUs)
			tStr := re.FindStringSubmatch(r[i])[2]
			t, _ := strconv.ParseFloat(tStr, 32)
			data["response_time"] = fmt.Sprintf("%f", (t / 1000))
		}

		// match run time in unit s
		match, _ = regexp.MatchString(reRunTimeS, r[i])
		if match {
			re := regexp.MustCompile(reRunTimeS)
			tStr := re.FindStringSubmatch(r[i])[2]
			t, _ := strconv.ParseFloat(tStr, 32)
			data["run_time"] = fmt.Sprintf("%f", (t * 1000))
		}
		// match run time in unit ms
		match, _ = regexp.MatchString(reRunTimeMs, r[i])
		if match {
			re := regexp.MustCompile(reRunTimeMs)
			tStr := re.FindStringSubmatch(r[i])[2]
			t, _ := strconv.ParseFloat(tStr, 32)
			data["run_time"] = fmt.Sprintf("%f", t)
		}
		// match run time in unit μs
		match, _ = regexp.MatchString(reRunTimeUs, r[i])
		if match {
			re := regexp.MustCompile(reRunTimeUs)
			tStr := re.FindStringSubmatch(r[i])[2]
			t, _ := strconv.ParseFloat(tStr, 32)
			data["run_time"] = fmt.Sprintf("%f", (t / 1000))
		}

		// match packet loss
		match, _ = regexp.MatchString(rePktLoss, r[i])
		if match {
			re := regexp.MustCompile(rePktLoss)
			data["packet_loss"] = re.FindStringSubmatch(r[i])[1]
		}

		// save used path (default or interactive) for later user display
		if pathNext {
			path = strings.TrimSpace(r[i])
		}
		match, _ = regexp.MatchString(reUPath, r[i])
		pathNext = match

		// evaluate error message potential
		match1, _ := regexp.MatchString(reErr1, r[i])
		match2, _ := regexp.MatchString(reErr2, r[i])
		match3, _ := regexp.MatchString(reErr3, r[i])

		if match1 {
			re := regexp.MustCompile(reErr1)
			err = re.FindStringSubmatch(r[i])[1]
			//log.Info("match1", "err", err)
		} else if match2 {
			re := regexp.MustCompile(reErr2)
			err = re.FindStringSubmatch(r[i])[1]
		} else if match3 {
			re := regexp.MustCompile(reErr3)
			err = re.FindStringSubmatch(r[i])[1]
		}
	}
	log.Info("app response", "data", data)

	//log.Info("print parsed result", "error", err)
	//log.Info("print parsed result", "path", path)

	d.ResponseTime, _ = strconv.ParseFloat(data["response_time"], 32)
	d.RunTime, _ = strconv.ParseFloat(data["run_time"], 32)
	d.PktLoss, _ = strconv.Atoi(data["packet_loss"])
	d.Error = err
	d.Path = path
	d.CmdOutput = resp // pipe log output to render in display later
}

// GetEchoByTimeHandler request the echo results stored since provided time.
func GetEchoByTimeHandler(w http.ResponseWriter, r *http.Request, active bool, srcpath string) {
	r.ParseForm()
	since := r.PostFormValue("since")
	log.Info("Requesting data since", "timestamp", since)
	// find undisplayed test results
	echoResults, err := model.ReadEchoItemsSince(since)
	if CheckError(err) {
		returnError(w, err)
		return
	}
	log.Debug("Requested data:", "echoResults", echoResults)

	echosJSON, err := json.Marshal(echoResults)
	if CheckError(err) {
		returnError(w, err)
		return
	}
	jsonBuf := []byte(`{ "graph": ` + string(echosJSON))
	json := []byte(`, "active": ` + strconv.FormatBool(active))
	jsonBuf = append(jsonBuf, json...)
	jsonBuf = append(jsonBuf, []byte(`}`)...)

	// ensure % if any, is escaped correctly before writing to printf formatter
	fmt.Fprintf(w, strings.Replace(string(jsonBuf), "%", "%%", -1))
}

// WriteEchoCsv appends the echo data in csv-format to srcpath.
func WriteEchoCsv(echo *model.EchoItem, srcpath string) {
	// newfile name for every day
	dataFileEcho := "data/echo-" + time.Now().Format("2006-01-02") + ".csv"
	bwdataPath := path.Join(srcpath, dataFileEcho)
	// write headers if file is new
	writeHeader := false
	if _, err := os.Stat(dataFileEcho); os.IsNotExist(err) {
		writeHeader = true
	}
	// open/create file
	f, err := os.OpenFile(bwdataPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if CheckError(err) {
		return
	}
	w := csv.NewWriter(f)
	// export headers if this is a new file
	if writeHeader {
		headers := echo.GetHeaders()
		w.Write(headers)
	}
	values := echo.ToSlice()
	w.Write(values)
	w.Flush()
}
