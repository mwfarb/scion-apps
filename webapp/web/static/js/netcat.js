// https://github.com/gorilla/websocket/blob/master/examples/chat/home.html
// https://medium.com/learning-the-go-programming-language/streaming-io-in-go-d93507931185
// https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go
// https://stackoverflow.com/questions/8309648/netcat-streaming-using-udp
// https://stackoverflow.com/questions/47231085/go-stdout-stream-from-async-command

// netcat -l -local 1-ff00:0:112,[127.0.0.2] 4141
// netcat -local 1-ff00:0:111,[127.0.0.1] 1-ff00:0:112,[127.0.0.2] 4141

$(document).ready(function() {

    var conn;
    var msg = document.getElementById("nc-msg");
    var log = document.getElementById("nc-log");

    function appendLog(item) {
        var doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;
        log.appendChild(item);
        if (doScroll) {
            log.scrollTop = log.scrollHeight - log.clientHeight;
        }
    }

    var formCheck = document.getElementById("nc-form");
    // formCheck.onSubmit = function() {
    // if (!conn) {
    // return false;
    // }
    // if (!msg.value) {
    // return false;
    // }
    // conn.send(msg.value);
    // msg.value = "";
    // return false;
    // };

    if (window["WebSocket"]) {
        conn = new WebSocket("ws://" + document.location.host + "/ws");
        conn.onclose = function(evt) {
            var item = document.createElement("div");
            item.innerHTML = "<b>Connection closed.</b>";
            appendLog(item);
        };
        conn.onmessage = function(evt) {
            var messages = evt.data.split('\n');
            for (var i = 0; i < messages.length; i++) {
                var item = document.createElement("div");
                item.innerText = messages[i];
                appendLog(item);
            }
        };
    } else {
        var item = document.createElement("div");
        item.innerHTML = "<b>Your browser does not support WebSockets.</b>";
        appendLog(item);
    }

});
