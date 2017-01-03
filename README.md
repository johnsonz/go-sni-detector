# go-checksni

[![Build Status](https://travis-ci.org/johnsonz/go-checksni.svg?branch=master)](https://travis-ci.org/johnsonz/go-checksni) [![GPLv3 License](https://img.shields.io/badge/license-GPLv3-blue.svg)](https://github.com/johnsonz/go-checksni/blob/master/LICENSE)
============

## 说明
用于扫描SNI服务器，请将待测试的ip段放到sniip.txt文件，支持以下ip格式：

1. xxx.xxx.xxx.xxx
2. xxx.xxx.xxx.xxx/xx
3. xxx.xxx.xxx.xxx-xxx.xxx.xxx.xxx
4. xxx.xxx.xxx.xxx-xxx.
5. xxx.-xxx.

## 下载地址
https://github.com/johnsonz/go-checksni/releases

## 编译
gox https://github.com/mitchellh/gox

## 配置说明
`"concurrency":1000` 并发线程数，可根据自己的硬件配置调整

`server_name` 用于测试SNI服务器的域名
