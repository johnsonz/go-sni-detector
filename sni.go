package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
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

func init() {
	parseConfig()
	loadCertPem()
}
func main() {

	usage()
	fmt.Printf("%v\n\n", config)
	time.Sleep(5 * time.Second)

	createFile()

	var ips []string
	status := getStatus()
	status = strings.Replace(strings.Replace(status, "\n", "", -1), "\r", "", -1)

	var lastOKIP []string
	for _, ip := range getLastOkIP() {
		lastOKIP = append(lastOKIP, ip.Address)
	}

	if config.SoftMode {
		totalips = make(chan string, config.Concurrency*10)
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
	for ip := range jobs {
		done <- true
		go checkIP(ip, done)
	}
	for i := 0; i < cap(done); i++ {
		done <- true
	}
	t1 := time.Now()
	cost := int(t1.Sub(t0).Seconds())
	rawipnum, jsonipnum := getJSONIP()
	updateConfig(config.IsOverride)
	write2File("true", statusFileName)
	fmt.Printf("\ntime: %ds, ok ip count: %d, matched ip with delay(%dms) count: %d\n\n", cost, rawipnum, config.Delay, jsonipnum)
	fmt.Scanln()
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
func checkIP(ip string, done chan bool) {
	defer func() {
		<-done
		appendIP2File(IP{Address: ip, Delay: 0, HostName: "-"}, sniNoFileName)
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
	err = json.Unmarshal(conf, &config)
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
		case Info, Warning, Debug:
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
    "delay":%d,
    "server_name":[
        %s
    ],
    "sort_by_delay":true,
	"always_check_all_ip":false
}`, config.Concurrency, config.Timeout, config.Delay,
				fmt.Sprint("\"", strings.Join(config.ServerName, "\",\n        \""), "\"")), configFileName)
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

func usage() {
	flag.Usage = func() {
		fmt.Printf(`
Usage: go-sni-detector [COMMANDS] [VARS]

SUPPORT COMMANDS:
	-h, --help          %s
	-a, --allhostname   %s
	-r, --override      %s

SUPPORT VARS:
	-i, --snifile           %s
	-o, --outputfile        %s
	-j, --jsonfile          %s
	-c, --concurrency       %s (default: %d)
	-t, --timeout           %s (default: %dms)
	-ht, --handshaketimeout %s (default: %dms)
	-d, --delay             %s (default: %dms)
	-s, --servername        %s (default: %s)
				`, helpMsg, allHostnameMsg, overrideMsg, sniFileMsg, outputFileMsg, jsonFileMsg, concurrencyMsg, config.Concurrency, timeoutMsg, config.Timeout, handshakeTimeoutMsg, config.HandshakeTimeout, delayMsg, config.Delay, serverNameMsg, strings.Join(config.ServerName, ", "))
	}
	var (
		outputAllHostname bool
		sniFile           string
		outputFile        string
		jsonFile          string
		concurrency       int
		timeout           int
		handshaketimeout  int
		delay             int
		serverNames       string
		isOverride        bool
	)

	flag.BoolVar(&outputAllHostname, "a", false, allHostnameMsg)
	flag.BoolVar(&outputAllHostname, "allhostname", false, allHostnameMsg)
	flag.StringVar(&sniFile, "i", sniIPFileName, sniFileMsg)
	flag.StringVar(&sniFile, "snifile", sniIPFileName, sniFileMsg)
	flag.StringVar(&outputFile, "o", sniResultFileName, outputFileMsg)
	flag.StringVar(&outputFile, "outputfile", sniResultFileName, outputFileMsg)
	flag.StringVar(&jsonFile, "j", sniJSONFileName, jsonFileMsg)
	flag.StringVar(&jsonFile, "jsonfile", sniJSONFileName, jsonFileMsg)
	flag.IntVar(&concurrency, "c", config.Concurrency, concurrencyMsg)
	flag.IntVar(&concurrency, "concurrency", config.Concurrency, concurrencyMsg)
	flag.IntVar(&timeout, "t", config.Timeout, timeoutMsg)
	flag.IntVar(&timeout, "timeout", config.Timeout, timeoutMsg)
	flag.IntVar(&handshaketimeout, "ht", config.HandshakeTimeout, handshakeTimeoutMsg)
	flag.IntVar(&handshaketimeout, "handshaketimeout", config.HandshakeTimeout, handshakeTimeoutMsg)
	flag.IntVar(&delay, "d", config.Delay, delayMsg)
	flag.IntVar(&delay, "delay", config.Delay, delayMsg)
	flag.StringVar(&serverNames, "s", strings.Join(config.ServerName, ", "), serverNameMsg)
	flag.StringVar(&serverNames, "servername", strings.Join(config.ServerName, ", "), serverNameMsg)
	flag.BoolVar(&isOverride, "r", false, overrideMsg)
	flag.BoolVar(&isOverride, "override", false, overrideMsg)

	flag.Set("logtostderr", "true")
	flag.Parse()

	sniIPFileName = sniFile
	sniResultFileName = outputFile
	sniJSONFileName = jsonFile

	if !isFileExist(sniFile) {
		fmt.Printf("file %s not found.\n", sniIPFileName)
		return
	}

	config.OutputAllHostname = outputAllHostname
	config.Concurrency = concurrency
	config.Timeout = timeout
	config.HandshakeTimeout = handshaketimeout
	config.Delay = delay
	sNs := strings.Split(serverNames, ",")
	for i, sn := range sNs {
		sNs[i] = strings.TrimSpace(sn)
	}
	config.ServerName = sNs
	config.IsOverride = isOverride
}
func getInputFromCommand() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.Replace(strings.Replace(input, "\n", "", -1), "\r", "", -1)
	return input
}
