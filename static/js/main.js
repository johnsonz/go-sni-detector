$(document).ready(function(){

    var end =false;
    var ws;
    var index=1;
    $("#btn-start").click(function(){
        $(this).attr("disabled","disabled");
        $(this).val("正在扫描");
        index=1;
        //if(!ws){return;}
        ws = new WebSocket("ws://127.0.0.1:8888/scan");
        ws.onopen = function(evt) {
            var file = document.getElementById('file').files[0];
            if(file==undefined||file==''||file==null){
                ws.send("nofile");
            }else{
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
            var data=evt.data
            console.log(data);
            if(data=="done"){
                //ws.close();
                $("#btn-start").removeAttr("disabled");
                $("#btn-start").val("开始");
            }else{
                $("#t-ips tr:last").after("<tr><td>"+index+"</td><td>"+evt.data+"</td><td>"+evt.data+"</td><td>"+evt.data+"</td></tr>");
                index++;
            }
        }
        ws.onerror = function(evt) {
            console.log("error",evt.data);
        }
        ws.onclose = function() {
            console.log("close");
            $("#btn-start").removeAttr("disabled");
            $("#btn-start").val("开始");
        }
    });

    $("#btn-config-update").click(function(){

    });

    $("#btn-config-reset").click(function(evt){

    });

});
