package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

//SNI config
type SNI struct {
	Concurrency       int      `json:"concurrency"`
	Timeout           int      `json:"timeout"`
	HandshakeTimeout  int      `json:"handshake_timeout"`
	Delay             int      `json:"delay"`
	ServerName        []string `json:"server_name"`
	SortByDelay       bool     `json:"sort_by_delay"`
	AlwaysCheck       bool     `json:"always_check_all_ip"`
	SoftMode          bool     `json:"soft_mode"`
	OutputAllHostname bool
	IsOverride        bool
}

//Result return messge to client
type Result struct {
	Status  bool
	Message string
}

//ScanResult return messge to client
type ScanResult struct {
	IPAddress string
	IsOkIIP   bool
	Delay     int
	Hostname  string
	Number    int
}

const (
	configFileName      string = "sni.json"
	configUserFileName  string = "sni.user.json"
	certFileName        string = "cacert.pem"
	sniIPSourceFileName string = "sniip.txt"
	sniIPTempFileName   string = "sniip_temp.txt"
	sniResultFileName   string = "sniip_ok.txt"
	sniNoFileName       string = "sniip_no.txt"
	sniJSONFileName     string = "ip.txt"
)

var (
	sniIPFileName = "sniip.txt"
)

//custom log level
const (
	Info = iota
	Warning
	Debug
	Error
)

var config SNI
var certPool *x509.CertPool
var tlsConfig *tls.Config
var dialer net.Dialer
var totalips chan string
var scanResult chan ScanResult
var upgrader = websocket.Upgrader{}
var templates *template.Template
var totalScanned int

func init() {
	config = parseConfig(configUserFileName)
	loadCertPem()

	templates = template.Must(template.New("templates").ParseGlob("./templates/*.gtpl"))
}
func main() {

	createFile()

	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/scan", scanHandler)
	http.HandleFunc("/config/update", updateConfigHandler)
	http.HandleFunc("/config/reset", resetConfigHandler)
	http.HandleFunc("/file/upload", fileUploadHandler)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	if err := http.ListenAndServe(":8888", nil); err != nil {
		checkErr("ListenAndServe error: ", err, Error)
	}
}

//Load cacert.pem
func loadCertPem() {
	certpem, err := ioutil.ReadFile(certFileName)
	checkErr(fmt.Sprintf("read pem file %s error: ", certFileName), err, Error)
	certPool = x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certpem) {
		checkErr(fmt.Sprintf("load pem file %s error: ", certFileName), errors.New("load pem file error"), Error)
	}
}
func checkIP(ip string, done chan bool, conn *websocket.Conn, msgType int) {
	defer func() {
		<-done
		appendIP2File(IP{Address: ip, Delay: 0, HostName: "-"}, sniNoFileName)
		totalScanned++
		scanResult <- ScanResult{ip, true, 0, "-", totalScanned}
	}()
	delays := make([]int, len(config.ServerName))
	dialer = net.Dialer{
		Timeout:   time.Millisecond * time.Duration(config.Timeout),
		KeepAlive: 0,
		DualStack: false,
	}
Next:
	for i, server := range config.ServerName {
		conn, err := dialer.Dial("tcp", net.JoinHostPort(ip, "443"))
		if err != nil {
			checkErr(fmt.Sprintf("%s dial error: ", ip), err, Debug)
			return
		}
		defer conn.Close()

		tlsConfig = &tls.Config{
			RootCAs:            certPool,
			InsecureSkipVerify: false,
			ServerName:         server,
		}

		t0 := time.Now()
		tlsClient := tls.Client(conn, tlsConfig)
		tlsClient.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(config.HandshakeTimeout)))
		err = tlsClient.Handshake()

		if err != nil {
			checkErr(fmt.Sprintf("%s handshake error: ", ip), err, Debug)
			return
		}
		defer tlsClient.Close()
		t1 := time.Now()
		delays[i] = int(t1.Sub(t0).Seconds() * 1000)
		if tlsClient.ConnectionState().PeerCertificates == nil {
			checkErr(fmt.Sprintf("%s peer certificates error: ", ip), errors.New("peer certificates is nil"), Debug)
			return
		}

		//peerCertSubject := tlsClient.ConnectionState().PeerCertificates[0].Subject
		DNSNames := tlsClient.ConnectionState().PeerCertificates[0].DNSNames
		//commonName := peerCertSubject.CommonName
		for _, DNSName := range DNSNames {
			if DNSName == server || DNSName == strings.Replace(server, "www", "*", -1) {
				continue Next
			}
		}
		return
	}
	sum := 0
	for _, d := range delays {
		sum += d
	}
	delay := sum / len(delays)

	hostname := "-"
	addr, err := net.LookupAddr(ip)
	if err == nil {
		if len(addr) > 0 {
			if config.OutputAllHostname {
				hostname = strings.Join(addr, "|")
			} else {
				hostname = addr[0]
			}
		}
	}

	checkErr(fmt.Sprintf("%s %dms %s, sni ip, recorded.", ip, delay, hostname), errors.New(""), Info)

	appendIP2File(IP{Address: ip, Delay: delay, HostName: hostname}, sniResultFileName)
	totalScanned++
	scanResult <- ScanResult{ip, true, delay, hostname, totalScanned}
}

//Parse config file
func parseConfig(filename string) (conf SNI) {
	data, err := ioutil.ReadFile(filename)
	checkErr("read config file error: ", err, Error)

	var lines []string
	for _, line := range strings.Split(strings.Replace(string(data), "\r\n", "\n", -1), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "//") && line != "" {
			lines = append(lines, line)
		}
	}

	var b bytes.Buffer
	for i, line := range lines {
		if len(lines)-1 > i {
			nextLine := lines[i+1]
			if nextLine == "]" || nextLine == "]," || nextLine == "}" || nextLine == "}," {
				if strings.HasSuffix(line, ",") {
					line = strings.TrimSuffix(line, ",")
				}
			}
		}
		b.WriteString(line)
	}

	err = json.Unmarshal(b.Bytes(), &conf)
	checkErr("parse config file error: ", err, Error)
	return conf
}

//Create files if they donnot exist, or truncate them.
func createFile() {
	if !isFileExist(sniResultFileName) {
		_, err := os.Create(sniResultFileName)
		checkErr(fmt.Sprintf("create file %s error: ", sniResultFileName), err, Error)
	}
	if !isFileExist(sniJSONFileName) {
		_, err := os.Create(sniJSONFileName)
		checkErr(fmt.Sprintf("create file %s error: ", sniJSONFileName), err, Error)
	}
	if !isFileExist(sniNoFileName) {
		_, err := os.Create(sniNoFileName)
		checkErr(fmt.Sprintf("create file %s error: ", sniNoFileName), err, Error)
	}
}

//CheckErr checks given error
func checkErr(messge string, err error, level int) {
	if err != nil {
		switch level {
		case Info:
			color.Set(color.FgGreen)
			defer color.Unset()
			glog.Infoln(messge, err)
		case Warning, Debug:
			glog.Infoln(messge, err)
		case Error:
			glog.Fatalln(messge, err)
		}
	}
}

//Whether file exists.
func isFileExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}
func getJSONIP() (rawipnum, jsonipnum int) {
	var rawiplist, jsoniplist []string
	okIPs := getLastOkIP()
	if config.SortByDelay {
		sort.Sort(ByDelay{IPs(okIPs)})
	}
	for _, ip := range okIPs {
		rawiplist = append(rawiplist, fmt.Sprintf("%s %dms %s", ip.Address, ip.Delay, ip.HostName))
		if ip.Delay <= config.Delay {
			jsoniplist = append(jsoniplist, ip.Address)
		}
	}

	ipstr := strings.Join(rawiplist, "\n")
	rawipnum = len(rawiplist)
	write2File(ipstr, sniResultFileName)
	jsonip := strings.Join(jsoniplist, "|")
	jsonip += "\n\n\n"
	jsonip += `"`
	jsonip += strings.Join(jsoniplist, `","`)
	jsonip += `"`
	jsonipnum = len(jsoniplist)
	write2File(jsonip, sniJSONFileName)
	return
}

//append ip to related file
func appendIP2File(ip IP, filename string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	checkErr(fmt.Sprintf("open file %s error: ", filename), err, Error)
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s %dms %s\n", ip.Address, ip.Delay, ip.HostName))
	checkErr(fmt.Sprintf("append ip to file %s error: ", filename), err, Error)
	f.Close()
}

//write ip to related file
func write2File(str string, filename string) {
	err := ioutil.WriteFile(filename, []byte(str), 0755)
	checkErr(fmt.Sprintf("write ip to file %s error: ", filename), err, Error)
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	templates = template.Must(template.New("templates").ParseGlob("./templates/*.gtpl"))
	templates.ExecuteTemplate(w, "main.gtpl", config)
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	totalScanned = 0
	conn, err := upgrader.Upgrade(w, r, nil)
	checkErr("websocket conn error: ", err, Error)
	defer conn.Close()
	mt, data, err := conn.ReadMessage()
	checkErr("websocket read error: ", err, Warning)
	content := string(data[:])
	fmt.Println("content from client: ", content)
	if content == "start" {
		var ips []string
		scanResult = make(chan ScanResult, config.Concurrency)

		if config.SoftMode {
			totalips = make(chan string, config.Concurrency)
			go func() {
				getSNIIPQueue(&totalips)
				close(totalips)
			}()
		} else {
			ips = getSNIIP()
			err = os.Truncate(sniNoFileName, 0)
			checkErr(fmt.Sprintf("truncate file %s error: ", sniNoFileName), err, Error)
			ips = getSNIIP()
			ips = getDifference(getSNIIP(), getLastNoIP())
		}
		err = os.Truncate(sniResultFileName, 0)
		checkErr(fmt.Sprintf("truncate file %s error: ", sniResultFileName), err, Error)
		jobs := make(chan string, config.Concurrency)
		done := make(chan bool, config.Concurrency)

		//check all sni ip begin
		t0 := time.Now()
		go func() {
			if config.SoftMode {
				for ip := range totalips {
					jobs <- ip
				}
			} else {
				for _, ip := range ips {
					jobs <- ip
				}
			}
			close(jobs)
		}()
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			// defer wg.Done()
			for r := range scanResult {
				v, _ := json.Marshal(r)
				conn.WriteMessage(mt, v)
			}
		}()

		for ip := range jobs {
			done <- true
			go checkIP(ip, done, conn, mt)
		}

		for i := 0; i < cap(done); i++ {
			done <- true
		}
		// close(scanResult)
		t1 := time.Now()
		cost := int(t1.Sub(t0).Seconds())
		var msg = ""
		m := math.Mod(float64(cost), 60)
		msg += strconv.FormatFloat(m, 'f', 0, 64) + "秒"
		if f1 := cost / 60; f1 > 0 {
			m := math.Mod(float64(f1), 60)
			msg = strconv.FormatFloat(m, 'f', 0, 64) + "分" + msg
			if f2 := f1 / 60; f2 > 0 {
				m := math.Mod(float64(f2), 60)
				msg = strconv.FormatFloat(m, 'f', 0, 64) + "时" + msg
				if f3 := f2 / 24; f3 > 0 {
					m := math.Mod(float64(f3), 24)
					msg = strconv.FormatFloat(m, 'f', 0, 64) + "天" + msg
				}
			}
		}
		fmt.Println(msg)
		// wg.Wait()
		v, _ := json.Marshal(Result{true, msg})
		conn.WriteMessage(mt, v)
	}
}
func updateConfigHandler(w http.ResponseWriter, r *http.Request) {
	concurrency := r.FormValue("concurrency")
	timeout := r.FormValue("timeout")
	handshaketimeout := r.FormValue("handshaketimeout")
	delay := r.FormValue("delay")
	servername := r.FormValue("servername")
	sort := r.FormValue("sort")
	softmode := r.FormValue("softmode")

	c, err := strconv.Atoi(concurrency)
	if err != nil {
		v, _ := json.Marshal(Result{false, "并发数只能为正整数"})
		fmt.Fprint(w, string(v))
		return
	}
	t, err := strconv.Atoi(timeout)
	if err != nil {
		v, _ := json.Marshal(Result{false, "超时时间只能为正整数"})
		fmt.Fprint(w, string(v))
		return
	}
	ht, err := strconv.Atoi(handshaketimeout)
	if err != nil {
		v, _ := json.Marshal(Result{false, "握手时间只能为正整数"})
		fmt.Fprint(w, string(v))
		return
	}
	d, err := strconv.Atoi(delay)
	if err != nil {
		v, _ := json.Marshal(Result{false, "延迟只能为正整数"})
		fmt.Fprint(w, string(v))
		return
	}
	servername = regexp.MustCompile(`\s{1,}`).ReplaceAllString(servername, " ")
	sn := strings.Split(servername, " ")
	if len(sn) < 1 {
		v, _ := json.Marshal(Result{false, "请填写Sever Name"})
		fmt.Fprint(w, string(v))
		return
	}
	s, err := strconv.ParseBool(sort)
	if err != nil {
		v, _ := json.Marshal(Result{false, "按延迟排序参数错误"})
		fmt.Fprint(w, string(v))
		return
	}
	sm, err := strconv.ParseBool(softmode)
	if err != nil {
		v, _ := json.Marshal(Result{false, "Soft模式参数错误"})
		fmt.Fprint(w, string(v))
		return
	}
	config = SNI{
		Concurrency:      c,
		Timeout:          t,
		HandshakeTimeout: ht,
		Delay:            d,
		ServerName:       sn,
		SortByDelay:      s,
		SoftMode:         sm,
	}
	updateConfig(config)
	v, _ := json.Marshal(Result{true, "update config successfully"})
	fmt.Fprint(w, string(v))
}

func resetConfigHandler(w http.ResponseWriter, r *http.Request) {
	conf := parseConfig(configFileName)
	config = conf
	updateConfig(config)
	v, _ := json.Marshal(config)
	fmt.Fprint(w, string(v))
}

func updateConfig(config SNI) {
	write2File(
		fmt.Sprintf(`{
		"concurrency":%d,
		"timeout":%d,
		"handshake_timeout":%d,
		"delay":%d,
		"server_name":[
			%s
		],
		"sort_by_delay":%t,
		"soft_mode":%t
	 }`, config.Concurrency, config.Timeout, config.HandshakeTimeout, config.Delay,
			fmt.Sprint("\"", strings.Join(config.ServerName, "\",\n\t\t\t\""), "\""), config.SortByDelay, config.SoftMode), configUserFileName)

	fmt.Println("update config successfully")
}

func fileUploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	checkErr("parse form error: ", err, Error)
	sniIPFileName = sniIPSourceFileName
	file, _, err := r.FormFile("file")
	var data []byte
	switch err {
	case nil:
		data, err = ioutil.ReadAll(file)
		if err != nil {
			checkErr("read file error: ", err, Error)
		}
		sniIPFileName = sniIPTempFileName
		err = ioutil.WriteFile(sniIPTempFileName, data, 0755)
		checkErr("write data to temp file error: ", err, Error)
	case http.ErrMissingFile:

		checkErr("no file uploaded error: ", errors.New(""), Warning)
	default:
		checkErr("upload file error: ", errors.New(""), Error)
	}

}
