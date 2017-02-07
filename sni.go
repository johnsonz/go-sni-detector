package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
)

//SNI config
type SNI struct {
	Concurrency      int `json:"concurrency"`
	Timeout          int `json:"timeout"`
	HandshakeTimeout int
	ServerName       []string `json:"server_name"`
}

const (
	sniIPFileName     string = "sniip.txt"
	sniResultFileName string = "sniip_output.txt"
	sniJSONFileName   string = "ip.txt"
	configFileName    string = "sni.json"
	certFileName      string = "cacert.pem"
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

func init() {
	fmt.Println("initial...")
	parseConfig()
	config.HandshakeTimeout = config.Timeout
	loadCertPem()
	createFile()
}
func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()

	ips := getSNIIP()
	lastOKIP := getLastOkIP()
	ips = append(lastOKIP, ips...)
	err := os.Truncate(sniResultFileName, 0)
	checkErr(fmt.Sprintf("truncate file %s error: ", sniResultFileName), err, Error)
	jobs := make(chan string, config.Concurrency)
	done := make(chan bool, config.Concurrency)

	//check all sni ip begin
	t0 := time.Now()
	go func() {
		for _, ip := range ips {
			jobs <- ip
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
	iplist := getLastOkIP()
	ipstr := strings.Join(iplist, "\n")
	writeIP2File(ipstr, sniResultFileName)
	jsonip := strings.Join(iplist, "|")
	jsonip += "\n\n\n"
	jsonip += `"`
	jsonip += strings.Join(iplist, `","`)
	jsonip += `"`
	writeIP2File(jsonip, sniJSONFileName)

	fmt.Printf("\ntime: %ds, ok ip count: %d\n\n", cost, len(iplist))
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
	}()
	dialer = net.Dialer{
		Timeout:   time.Millisecond * time.Duration(config.Timeout),
		KeepAlive: 0,
		DualStack: false,
	}
Next:
	for _, server := range config.ServerName {
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

		//t0 := time.Now()
		tlsClient := tls.Client(conn, tlsConfig)
		tlsClient.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(config.HandshakeTimeout)))
		err = tlsClient.Handshake()

		if err != nil {
			checkErr(fmt.Sprintf("%s handshake error: ", ip), err, Debug)
			return
		}
		defer tlsClient.Close()
		//t1 := time.Now()

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
	appendIP2File(ip, sniResultFileName)
}

//Parse config file
func parseConfig() {
	conf, err := ioutil.ReadFile(configFileName)
	checkErr("read config file error: ", err, Error)
	err = json.Unmarshal(conf, &config)
	checkErr("parse config file error: ", err, Error)
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

//Whether file exists.
func isFileExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

//get all sni ip range from sniip.txt file
func getSNIIPRange() []string {
	m := make(map[string]string)
	var ipRanges []string
	bytes, err := ioutil.ReadFile(sniIPFileName)
	checkErr(fmt.Sprintf("read file %s error: ", sniIPFileName), err, Error)

	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines {
		line = strings.Replace(line, "\r", "", -1)
		if len(line) > 7 {
			m[line] = line
		}
	}
	for _, v := range m {
		ipRanges = append(ipRanges, v)
	}

	return ipRanges
}

/**
  Parse sni ip range, support the following formats:
  1. xxx.xxx.xxx.xxx
  2. xxx.xxx.xxx.xxx/xx
  3. xxx.xxx.xxx.xxx-xxx.xxx.xxx.xxx
  4. xxx.xxx.xxx.xxx-xxx.
  5. xxx.-xxx.
*/
func parseSNIIPRange(ipRange string) []string {
	var ips []string
	if strings.Contains(ipRange, "/") {
		//CIDR: https://zh.wikipedia.org/wiki/%E6%97%A0%E7%B1%BB%E5%88%AB%E5%9F%9F%E9%97%B4%E8%B7%AF%E7%94%B1
		ip, ipNet, err := net.ParseCIDR(ipRange)
		checkErr(fmt.Sprintf("parse CIDR %s error: ", ipRange), err, Error)

		for iptmp := ip.Mask(ipNet.Mask); ipNet.Contains(iptmp); inc(iptmp) {
			ips = append(ips, iptmp.String())
		}
		// remove network address and broadcast address
		return ips[1 : len(ips)-1]
	} else if strings.Contains(ipRange, "-") {
		startIP := ipRange[:strings.Index(ipRange, "-")]
		endIP := ipRange[strings.Index(ipRange, "-")+1:]
		if strings.HasSuffix(startIP, ".") {
			switch strings.Count(startIP, ".") {
			case 1:
				startIP += "0.0.0"
			case 2:
				startIP += "0.0"
			case 3:
				startIP += "0"
			}
		}
		if strings.HasSuffix(endIP, ".") {
			switch strings.Count(endIP, ".") {
			case 1:
				endIP += "255.255.255"
			case 2:
				endIP += "255.255"
			case 3:
				endIP += "255"
			}
		}
		sIP := net.ParseIP(startIP)
		eIP := net.ParseIP(endIP)

		for ip := sIP; bytes.Compare(ip, eIP) <= 0; inc(ip) {
			ips = append(ips, ip.String())

		}
	} else {
		ips = append(ips, ipRange)
	}

	return ips
}

//get all sni ip
func getSNIIP() []string {
	var ips []string
	ipRanges := getSNIIPRange()
	for _, ipRange := range ipRanges {
		ips = append(ips, parseSNIIPRange(ipRange)...)
	}

	return ips
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

//get last ip
func getLastOkIP() []string {
	m := make(map[string]string)
	var ips []string
	if isFileExist(sniResultFileName) {
		bytes, err := ioutil.ReadFile(sniResultFileName)
		checkErr(fmt.Sprintf("read file %s error: ", sniResultFileName), err, Error)
		lines := strings.Split(string(bytes), "\n")
		for _, line := range lines {
			if len(line) > 6 && len(line) < 16 {
				m[line] = line
			}
		}
	}
	for _, v := range m {
		ips = append(ips, v)
	}
	return ips
}

//append ip to related file
func appendIP2File(ip, filename string) {
	f, err := os.OpenFile(filename, os.O_APPEND, os.ModeAppend)
	checkErr(fmt.Sprintf("open file %s error: ", filename), err, Error)
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s\n", ip))
	checkErr(fmt.Sprintf("append ip to file %s error: ", filename), err, Error)
	f.Close()
}

//write ip to related file
func writeIP2File(ips string, filename string) {
	err := os.Truncate(filename, 0)
	checkErr(fmt.Sprintf("truncate file %s error: ", filename), err, Error)
	err = ioutil.WriteFile(filename, []byte(ips), 0755)
	checkErr(fmt.Sprintf("write ip to file %s error: ", filename), err, Error)
}
