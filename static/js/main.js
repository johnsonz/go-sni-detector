$(document).ready(function() {

    var end = false;
    var ws;
    var index = 1;
    var sninumber = 0;
    var totalnumber = 0;

    $('[data-toggle="tooltip"]').tooltip();

    $("#btn-start").click(function() {
        $(this).attr("disabled", "disabled");
        $(this).val("正在扫描");
        index = 1;
        sninumber = 0;
        totalnumber = 0;
        //if(!ws){return;}
        ws = new WebSocket("ws://127.0.0.1:8888/scan");
        ws.onopen = function(evt) {
            var file = document.getElementById('file').files[0];
            if (file == undefined || file == '' || file == null) {
                ws.send("nofile");
            } else {
                var reader = new FileReader();
                var rawData = new ArrayBuffer();
                reader.onload = function(e) {
                    rawData = e.target.result;
                    ws.send(rawData);
                }
                reader.readAsArrayBuffer(file);
            }

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
                $("#alert-result-status").html("已扫描：" + totalnumber + "，有效：" + sninumber);
                if (result.SNIIP) {
                    sninumber++;
                    $("#t-ips tr:last").after("<tr><td>" + index + "</td><td>" + result.IPAddress + "</td><td>" + result.Delay + "</td><td>" + result.Hostname + "</td></tr>");
                    index++;
                }
            }
        }
        ws.onerror = function(evt) {
            console.log("error", evt.data);
        }
        ws.onclose = function() {
            console.log("close");
            $("#btn-start").removeAttr("disabled");
            $("#btn-start").val("开始");
        }
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
});
