# go-sni-detector

[![Build Status](https://travis-ci.org/johnsonz/go-sni-detector.svg?branch=master)](https://travis-ci.org/johnsonz/go-sni-detector) [![GPLv3 License](https://img.shields.io/badge/license-GPLv3-blue.svg)](https://github.com/johnsonz/go-sni-detector/blob/master/LICENSE)
============

## 说明
用于扫描SNI服务器，sniip_output.txt中的延迟值为配置中指定的各server_name的延迟的平均值。

请将待测试的ip段放到sniip.txt文件，支持以下ip格式：

1. xxx.xxx.xxx.xxx
2. xxx.xxx.xxx.xxx/xx
3. xxx.xxx.xxx.xxx-xxx.xxx.xxx.xxx
4. xxx.xxx.xxx.xxx-xxx.
5. xxx.-xxx.

## 下载地址
[Latest release](https://github.com/johnsonz/go-sni-detector/releases)

## 编译
[gox](https://github.com/mitchellh/gox)

## 高级用法
支持命令，优先级高于配置文件，但配置文件不会被修改。
```
Usage: go-sni-detector [COMMAND] [VARS]

SUPPORT COMMANDS:
	-h, --help                   help message
	-a, --allhostname            lookup all hostname from ip, or lookup the first one by default

SUPPORT VARS:
	-i, --snifile<=path>         put your ip ranges into this file
	-o, --outputfile<=path>      output sni ip to this file
	-j, --jsonfile<=path>        output sni ip as json format to this file
	-c, --concurrency<=number>   concurrency
	-t, --timeout<=number>       timeout
	-d, --delay<=number>         delay
	-s, --servername<=string>    comma-separated server names
```

## 配置说明
`"concurrency":1000` 并发线程数，可根据自己的硬件配置调整

`"delay":1200` 扫描完成后，提取所有小于等于该延迟的ip

`"server_name"` 用于测试SNI服务器的域名
