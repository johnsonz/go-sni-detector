<!DOCTYPE html>
<html>

<head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title></title>

    <link rel="stylesheet" type="text/css" href="/static/bootstrap-3.3.7/css/bootstrap.min.css">
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
                    <li class="active"><a href="#">首页</a></li>
                    <li><a href="/tools">工具</a></li>
                    <li><a href="/help">帮助</a></li>
                </ul>
            </div>
            <!--/.nav-collapse -->
        </div>
    </nav>

    <div class="container">
        <div style="display:inline-table;width:70%;">
            <div>
                选择文件
                <input type="file" id="file" name="sni-file" style="display:inline;">
                <input type="button" style="" class="btn btn-primary" id="btn-start" value="开始"></input>
            </div>
            <div style="max-height:500px;overflow:auto;">
                <table class="table table-bordered table-striped" id="t-ips">
                    <tr>
                        <th>#</th>
                        <th>IP</th>
                        <th>延迟</th>
                        <th>主机名</th>
                    </tr>
                </table>
            </div>
            <div>
                <input type="button" class="btn btn-primary" value="全部导出为JSON" id="btn-export-json"/>
                <input type="button" class="btn btn-primary" value="全部导出为竖线分隔" id="btn-export-bar"/>
            </div>
        </div>
        <div class="form-inline" style="display:inline-table;width:25%">
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">并发数</span>
                <input type="number" class="form-control" id="basic-url" aria-describedby="basic-addon3" value="{{.Concurrency}}">
            </div>
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">超时时间</span>
                <input type="number" class="form-control" id="basic-url" aria-describedby="basic-addon3" value="{{.Timeout}}">
                <div class="input-group-addon">ms</div>
            </div>
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">握手超时时间</span>
                <input type="number" class="form-control" id="basic-url" aria-describedby="basic-addon3" value="{{.HandshakeTimeout}}">
                <div class="input-group-addon">ms</div>
            </div>
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">延迟</span>
                <input type="number" class="form-control" id="basic-url" aria-describedby="basic-addon3" value="{{.Delay}}">
                <div class="input-group-addon">ms</div>
            </div>
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">ServerName</span>
                <textarea type="text"  class="form-control" id="basic-url" aria-describedby="basic-addon3" >{{ range $index,$sn:=.ServerName}}{{$sn}} {{end}}</textarea>

            </div>
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">延迟排序</span> {{if .SortByDelay}}
                <input type="checkbox" value="" checked style="vertical-align: middle;margin-left:10px">{{else}}
                <input type="checkbox" value="" style="vertical-align: middle;margin-left:10px"> {{end}}
            </div>
            <br />
            <div class="input-group">
                <span class="input-group-addon" id="basic-addon3">soft模式</span> {{if .SoftMode}}
                <input type="checkbox" value="" checked style="vertical-align: middle;margin-left:10px"> {{else}}
                <input type="checkbox" value="" style="vertical-align: middle;margin-left:10px"> {{end}}
            </div>
            <br />
            <button type="submit" class="btn btn-primary" style="margin-top:10px;" id="btn-config-update">更新</button>
            <button type="submit" class="btn btn-primary" style="margin-left:20px;margin-top:10px;" id="btn-config-reset">重置为默认</button>
        </div>
    </div>

    <script src="/static/js/jquery-3.2.1.min.js"></script>
    <script src="/static/bootstrap-3.3.7/js/bootstrap.min.js"></script>
    <script src="/static/js/main.js"></script>
</body>

</html>
