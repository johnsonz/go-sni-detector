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
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
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

const (
	configFileName string = "sni.json"
	certFileName   string = "cacert.pem"
)

var (
	sniIPFileName     = "sniip.txt"
	sniResultFileName = "sniip_ok.txt"
	sniNoFileName     = "sniip_no.txt"
	sniJSONFileName   = "ip.txt"
	statusFileName    = ".status"
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
var okips chan string
var upgrader = websocket.Upgrader{}
var templates *template.Template

func init() {
	parseConfig()
	loadCertPem()

	templates = template.Must(template.New("templates").ParseGlob("./templates/*.gtpl"))
}
func main() {

	createFile()

	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/scan", scanHandler)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	if err := http.ListenAndServe(":8888", nil); err != nil {
		checkErr("ListenAndServe error: ", err, Error)
	}

	// rawipnum, jsonipnum := getJSONIP()
	// updateConfig(config.IsOverride)
	// write2File("true", statusFileName)
	// fmt.Printf("\ntime: %ds, ok ip count: %d, matched ip with delay(%dms) count: %d\n\n", cost, rawipnum, config.Delay, jsonipnum)
	// fmt.Scanln()
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
		okips <- ip
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
}

//Parse config file
func parseConfig() {
	conf, err := ioutil.ReadFile(configFileName)
	checkErr("read config file error: ", err, Error)

	var lines []string
	for _, line := range strings.Split(strings.Replace(string(conf), "\r\n", "\n", -1), "\n") {
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

	err = json.Unmarshal(b.Bytes(), &config)
	checkErr("parse config file error: ", err, Error)
}

func getStatus() string {
	status, err := ioutil.ReadFile(statusFileName)
	checkErr(fmt.Sprintf("read file %s error: ", statusFileName), err, Error)
	return string(status[:])
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
	if !isFileExist(statusFileName) {
		_, err := os.Create(statusFileName)
		checkErr(fmt.Sprintf("create file %s error: ", statusFileName), err, Error)
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

func updateConfig(isOverride bool) {
	if isOverride {
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
	"always_check_all_ip":%t,
	"soft_mode":%t
}`, config.Concurrency, config.Timeout, config.HandshakeTimeout, config.Delay,
				fmt.Sprint("\"", strings.Join(config.ServerName, "\",\n        \""), "\""), config.SortByDelay, config.AlwaysCheck, config.SoftMode), configFileName)
		fmt.Println("update sni.json successfully")
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
	err := os.Truncate(filename, 0)
	checkErr(fmt.Sprintf("truncate file %s error: ", filename), err, Error)
	err = ioutil.WriteFile(filename, []byte(str), 0755)
	checkErr(fmt.Sprintf("write ip to file %s error: ", filename), err, Error)
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	templates = template.Must(template.New("templates").ParseGlob("./templates/*.gtpl"))
	templates.ExecuteTemplate(w, "main.gtpl", config)
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	checkErr("websocket conn error: ", err, Error)
	defer conn.Close()
	mt, data, err := conn.ReadMessage()
	checkErr("websocket read error: ", err, Warning)
	content := string(data[:])
	if content == "" || content == "nofile" {
		var ips []string
		status := getStatus()
		status = strings.Replace(strings.Replace(status, "\n", "", -1), "\r", "", -1)

		okips = make(chan string, config.Concurrency)

		var lastOKIP []string
		for _, ip := range getLastOkIP() {
			lastOKIP = append(lastOKIP, ip.Address)
		}

		if config.SoftMode {
			totalips = make(chan string, config.Concurrency)
			go func() {
				for _, ip := range lastOKIP {
					totalips <- ip
				}
				getSNIIPQueue()
				close(totalips)
			}()

			goto Queue
		}

		if config.AlwaysCheck {
			ips = getSNIIP()
		} else {
			if status == "true" {
				fmt.Println("所有IP已扫描，5秒钟后将执行重新扫描。")
				for i := 1; i < 6; i++ {
					time.Sleep(time.Millisecond * 1000)
					fmt.Print("\r", i, "s")
				}
				err := os.Truncate(sniNoFileName, 0)
				checkErr(fmt.Sprintf("truncate file %s error: ", sniNoFileName), err, Error)
				ips = getSNIIP()
			} else {
				ips = getDifference(getSNIIP(), getLastNoIP())
			}
		}

		ips = append(lastOKIP, ips...)
	Queue:
		err := os.Truncate(sniResultFileName, 0)
		checkErr(fmt.Sprintf("truncate file %s error: ", sniResultFileName), err, Error)
		write2File("false", statusFileName)
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

		go func() {
			for ip := range okips {
				conn.WriteMessage(mt, []byte(ip))
			}
		}()

		for ip := range jobs {
			done <- true
			go checkIP(ip, done, conn, mt)
		}

		for i := 0; i < cap(done); i++ {
			done <- true
		}
		close(okips)
		t1 := time.Now()
		cost := int(t1.Sub(t0).Seconds())
		fmt.Println(cost, "ms")
		conn.WriteMessage(mt, []byte("done"))

	}
}
