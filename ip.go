package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
)

//IP struct
type IP struct {
	Address  string
	Delay    int
	HostName string
}

//IPs []IP
type IPs []IP

//Len return the length of []IP
func (ips IPs) Len() int {
	return len(ips)
}

//Swap swap two value of []IP
func (ips IPs) Swap(i, j int) {
	ips[i], ips[j] = ips[j], ips[i]
}

//ByDelay sort by delay
type ByDelay struct {
	IPs
}

//Less return false if the first value less than the second one
func (s ByDelay) Less(i, j int) bool {
	return s.IPs[i].Delay < s.IPs[j].Delay
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
		line = strings.TrimSpace(line)
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
func getLastOkIP() []IP {
	m := make(map[string]IP)
	var checkedip IP
	var ips []IP
	if isFileExist(sniResultFileName) {
		bytes, err := ioutil.ReadFile(sniResultFileName)
		checkErr(fmt.Sprintf("read file %s error: ", sniResultFileName), err, Error)
		lines := strings.Split(string(bytes), "\n")
		for _, line := range lines {
			ipinfo := strings.Split(line, " ")
			if len(ipinfo) == 2 || len(ipinfo) == 3 {
				checkedip.Address = ipinfo[0]
				delay, err := strconv.Atoi(ipinfo[1][:len(ipinfo[1])-2])
				checkErr("delay conversion failed: ", err, Warning)
				checkedip.Delay = delay
				hostname := "-"
				if len(ipinfo) == 3 {
					hostname = ipinfo[2]
				}
				checkedip.HostName = hostname
				m[ipinfo[0]] = checkedip
			}
		}
	}
	for _, v := range m {
		ips = append(ips, v)
	}
	return ips
}
