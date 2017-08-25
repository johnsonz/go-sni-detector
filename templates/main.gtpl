<!DOCTYPE html>
<html>

<head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title></title>

    <link rel="stylesheet" href="/static/bootstrap-3.3.7/css/bootstrap.min.css">
    <link rel="stylesheet" href="/static/bootstrap-3.3.7/css/bootstrap-theme.min.css">
    <link rel="stylesheet" href="/static/css/main.css">

</head>

<body>
    <nav class="navbar navbar-inverse navbar-static-top">
        <div class="container">
            <div class="navbar-header">
                <button type="button" class="navbar-toggle collapsed" data-toggle="collapse" data-target="#navbar" aria-expanded="false" aria-controls="navbar">
            <span class="sr-only">Toggle navigation</span>
            <span class="icon-bar"></span>
            <span class="icon-bar"></span>
            <span class="icon-bar"></span>
          </button>
                <a class="navbar-brand" href="#">SNI Detector</a>
            </div>
            <div id="navbar" class="collapse navbar-collapse">
                <ul class="nav navbar-nav">
                    <li class="active"><a href="javascript:void(0);" id="nav-main">首页</a></li>
                    <li><a href="javascript:void(0);" id="nav-tool">工具</a></li>
                    <li><a href="javascript:void(0);" id="nav-help">帮助</a></li>
                </ul>
            </div>
            <!--/.nav-collapse -->
        </div>
    </nav>

    <div class="container container-main">
        <div class="container-file">
            <div>
                选择文件
                <input type="file" id="file" name="sni-file">
                <input type="button" style="" class="btn btn-primary" id="btn-start" value="开始"></input>
            </div>
            <div>
                <div class="alert alert-success" role="alert" id="alert-result-status"></div>
                <div class="alert alert-danger" role="alert" id="alert-error"></div>
            </div>
            <div class="container-table">
                <table class="table table-bordered table-striped" id="t-ips">
                    <tr>
                        <th><input type='checkbox' class='cb-ip' id='cb-select-all' /></th>
                        <th>#</th>
                        <th>IP</th>
                        <th>延迟(ms)</th>
                        <th>主机名</th>
                    </tr>
                </table>
            </div>
            <div class="container-btn">
                <input type="button" class="btn btn-primary" value="全选" id="btn-select-all" />
                <input type="button" class="btn btn-primary" value="全不选" id="btn-unselect-all" />
                <input type="button" class="btn btn-primary" value="全选延迟小于{{.Delay}}的" id="btn-select-delay" />
            </div>
            <div class="btn-group">
                <button type="button" class="btn btn-primary dropdown-toggle" data-toggle="dropdown" aria-haspopup="true" aria-expanded="false">将所选导出为
                <span class="caret"></span>
              </button>
                <ul class="dropdown-menu">
                    <li><a id="btn-export-json" href="javascript:void(0);">JSON格式到剪贴板</a></li>
                    <li><a id="btn-export-bar" href="javascript:void(0);">竖线分隔格式到剪贴板</a></li>
                </ul>
            </div>
            <div>
                <div class="alert alert-success" role="alert" id="alert-copy-clipboard"></div>
            </div>
        </div>
        <div class="form-inline">
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">并发数</span>
                <input type="number" class="form-control" id="concurrency" aria-describedby="basic-addon3" value="{{.Concurrency}}">
            </div>
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">超时时间</span>
                <input type="number" class="form-control" id="timeout" aria-describedby="basic-addon3" value="{{.Timeout}}">
                <div class="input-group-addon">ms</div>
            </div>
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">握手超时时间</span>
                <input type="number" class="form-control" id="handshake-timeout" aria-describedby="basic-addon3" value="{{.HandshakeTimeout}}">
                <div class="input-group-addon">ms</div>
            </div>
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">延迟</span>
                <input type="number" class="form-control" id="delay" aria-describedby="basic-addon3" value="{{.Delay}}">
                <div class="input-group-addon">ms</div>
            </div>
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">ServerName</span>
                <textarea type="text" class="form-control" id="server-name" aria-describedby="basic-addon3" data-toggle="tooltip" data-placement="top" data-container="body" title="请以空格分隔">{{ range $index,$sn:=.ServerName}}{{$sn}} {{end}}</textarea>

            </div>
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">延迟排序</span> {{if .SortByDelay}}
                <input type="checkbox" id="sort-by-delay" checked style="vertical-align: middle;margin-left:10px">{{else}}
                <input type="checkbox" id="sort-by-delay" style="vertical-align: middle;margin-left:10px"> {{end}}
            </div>
            <br />
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">soft模式</span> {{if .SoftMode}}
                <input type="checkbox" id="soft-mode" checked style="vertical-align: middle;margin-left:10px"> {{else}}
                <input type="checkbox" id="soft-mode" style="vertical-align: middle;margin-left:10px"> {{end}}
            </div>
            <br />
            <button type="submit" class="btn btn-primary" id="btn-config-update">更新</button>
            <button type="submit" class="btn btn-primary" id="btn-config-reset">重置为默认</button>
            <div class="alert alert-success" role="alert" id="alert-config">更新成功！</div>
        </div>
    </div>
    <div class="container container-tool">
        <div>
            <div class="alert alert-success alert-copy-clipboard" role="alert"></div>
        </div>
        <div>
            <textarea class="" rows="8" id="tt-raw-data" placeholder="请把包含 IP 地址的文本粘贴到此处"></textarea>
            <textarea class="" rows="8" id="tt-output-data" placeholder="格式化后的 IP 地址将显示在此" readonly="readonly"></textarea>
        </div>
        <div class="container-o">
            <input type="checkbox" checked="checked" id="cb-one-line" style="">每个IP占一行</input>
        </div>
        <div class="container-c">
            <input type="button" class="btn btn-primary" id="btn-convert-json" value="转换为JSON格式"></input>
            <input type="button" class="btn btn-primary" id="btn-convert-bar" value="转换为为竖线分隔格式"></input>
        </div>
    </div>
    <div class="container container-help">
        <h1>go-sni-detector windows版本<h1>
            <p>
            <a href="https://travis-ci.org/johnsonz/go-sni-detector" target="_blank"><img alt="Build Status" src="https://travis-ci.org/johnsonz/go-sni-detector.svg?branch=master"/></a>
            <a href="https://github.com/johnsonz/go-sni-detector/blob/master/LICENSE" target="_blank"><img alt="GPLv3 License" src="https://img.shields.io/badge/license-GPLv3-blue.svg"/></a>
            </p>
        <h2>说明</h2>
        <p>
            go-sni-detector windows浏览器版本，用于扫描SNI服务器，扫描出的延迟值为配置中指定的各Server Name的延迟的平均值。
        </p>
        <p>
            项目使用了websocket，请参见<a href="http://caniuse.mojijs.com/Home/Html/item/key/websockets/index.html" target="_blank">Web Sockets浏览器兼容一览表</a>自行判断浏览器是否支持websocket。
        </p>
        <h2>使用方法</h2>
        <p>
            请先选择包含ip段的文件，如果未指定，则使用内置的ip段。
        </p>
        <h2>下载地址</h2>
        <p>
            <a href="https://github.com/johnsonz/go-sni-detector/releases" target="_blank">Latest release</a>
        </p>
        <h2>其它工具</h2>
        <p>
            扫描google ip工具：<a href="https://github.com/johnsonz/go-checkiptools" target="_blank">go-checkiptools</a>
        </p>
        <h2>问题</h2>
        <p>
            项目还在完善中，可能会有些小问题，欢迎到<a href="https://github.com/johnsonz/go-sni-detector" target="_blank">go-sni-detector</a>提issue。
        </p>
    </div>
    <script src="/static/js/jquery-3.2.1.min.js"></script>
    <script src="/static/bootstrap-3.3.7/js/bootstrap.min.js"></script>
    <script src="/static/js/main.js"></script>
</body>

</html>
