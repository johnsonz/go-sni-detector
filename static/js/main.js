$(document).ready(function() {

    var end = false;
    var ws;
    var index = 1;
    var sninumber = 0;
    var totalnumber = 0;

    $('[data-toggle="tooltip"]').tooltip();

    $("#btn-start").click(function() {
        uploadFile();
    });

    $("#btn-config-update").click(function() {
        $.post("/config/update", {
            concurrency: $.trim($("#concurrency").val()),
            timeout: $.trim($("#timeout").val()),
            handshaketimeout: $.trim($("#handshake-timeout").val()),
            delay: $.trim($("#delay").val()),
            servername: $.trim($("#server-name").val()),
            sort: $.trim($("#sort-by-delay").is(":checked")),
            softmode: $.trim($("#soft-mode").is(":checked")),
        }, function(data, status) {
            var result = $.parseJSON(data);
            if (result.Status) {
                $("#alert-config").html("更新成功！");
                $("#alert-config").removeClass("alert-danger").addClass("alert-success").show();
                $("#alert-config").fadeOut(7000);
            } else {
                $("#alert-config").html("更新失败！" + result.Message);
                $("#alert-config").removeClass("alert-success").addClass("alert-danger").show();
            }
        });
    });

    $("#btn-config-reset").click(function(evt) {
        $.getJSON("/config/reset", function(data) {
            $("#concurrency").val(data.concurrency);
            $("#timeout").val(data.timeout);
            $("#handshake-timeout").val(data.handshake_timeout);
            $("#delay").val(data.delay);
            $("#server-name").val(data.server_name.join(" "));
            $("#sort-by-delay").prop('checked', data.sort_by_delay);
            $("#soft-mode").prop('checked', data.soft_mode);

            $("#alert-config").html("重置成功！");
            $("#alert-config").removeClass("alert-danger").addClass("alert-success").show();
            $("#alert-config").fadeOut(7000);
        });
    });

    $("#btn-select-all").click(function() {
        $(".cb-ip").prop('checked', true);
    });

    $("#btn-unselect-all").click(function() {
        $(".cb-ip").prop('checked', false);
    });

    $("#btn-export-json").click(function() {
        var data = "";
        $('#t-ips tr').filter(':has(:checkbox:checked)').each(function() {
            data += '"' + $(this).find(".td-ip-addr").html() + '",';
        });
        if (data.length > 0) {
            data = data.substr(0, data.length - 1);
        }
        copyToClipboard(data);
        $("#alert-copy-clipboard").html("已复制到剪贴板！");
        $("#alert-copy-clipboard").show();
        $("#alert-copy-clipboard").fadeOut(7000);
    });

    $("#btn-export-bar").click(function() {
        var data = "";
        $('#t-ips tr').filter(':has(:checkbox:checked)').each(function() {
            data += $(this).find(".td-ip-addr").html() + '|';
        });
        if (data.length > 0) {
            data = data.substr(0, data.length - 1);
        }
        copyToClipboard(data);
        $("#alert-copy-clipboard").html("已复制到剪贴板！");
        $("#alert-copy-clipboard").show();
        $("#alert-copy-clipboard").fadeOut(7000);
    });
});

function copyToClipboard(data) {
    var temp = $("<input>");
    $("body").append(temp);
    temp.val(data).select();
    document.execCommand("copy");
    temp.remove();
}

function scan() {
    $("#btn-start").attr("disabled", "disabled");
    $("#btn-start").val("正在扫描");
    index = 1;
    sninumber = 0;
    totalnumber = 0;
    //if(!ws){return;}
    ws = new WebSocket("ws://127.0.0.1:8888/scan");
    ws.onopen = function(evt) {
        ws.send("start");
        $("#t-ips tr").nextAll().remove();
    }
    ws.onmessage = function(evt) {
        var data = evt.data
        console.log(data);
        var result = $.parseJSON(data);
        $("#alert-result-status").show();
        if (result.Status == true) {
            //ws.close();
            $("#btn-start").removeAttr("disabled");
            $("#btn-start").val("开始");
            $("#alert-result-status").html("扫描完成，共扫描：" + totalnumber + "，有效：" + sninumber + "，耗时：" + result.Message);
        } else {
            totalnumber = result.Number;
            if (result.IsOkIIP) {
                sninumber++;
                $("#t-ips tr:last").after("<tr><td><input type='checkbox' class='cb-ip' id=''/></td><td>" + index + "</td><td class='td-ip-addr'>" + result.IPAddress + "</td><td class='td-ip-delay'>" + result.Delay + "</td><td class='td-ip-hostname'>" + result.Hostname + "</td></tr>");
                index++;
            }
            $("#alert-result-status").html("已扫描：" + totalnumber + "，有效：" + sninumber);
        }
    }
    ws.onerror = function(evt) {
        console.log("error", evt.data);
        alert("出错了，请尝试刷新页面或重新启动。")
    }
    ws.onclose = function() {
        console.log("close");
        $("#btn-start").removeAttr("disabled");
        $("#btn-start").val("开始");
    }
}

function uploadFile() {
    $("#btn-start").val("正在处理数据");
    var data = new FormData();
    data.append('file', $('#file')[0].files[0]);
    $.ajax({
        url: '/file/upload',
        async: true,
        type: 'POST',
        cache: false,
        data: data,
        processData: false,
        contentType: false,
    }).done(function(res) {
        scan();
    }).fail(function(res) {
        console.log("处理数据错误");
    });
}

function sort() {
    var ips = new Array();
    $('#t-ips tr').each(function() {
        var addr = $(this).find(".td-ip-addr").html();
        var delay = $(this).find(".td-ip-delay").html();
        var hostname = $(this).find(".td-ip-hostname").html();
        if (addr == undefined && delay == undefined && hostname == undefined) {
            return true;
        }
        ips.push(new Result(addr, parseInt(delay, 10), hostname));
    });

    ips.sort(function(a, b) {
        return a.delay < b.delay ? 1 : -1;
    });
}

function Result(addr, delay, hostname) {
    this.addr = addr;
    this.delay = delay;
    this.hostname = hostname;
}
