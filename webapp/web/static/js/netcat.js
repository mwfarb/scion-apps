// https://github.com/gorilla/websocket/blob/master/examples/chat/home.html
// https://medium.com/learning-the-go-programming-language/streaming-io-in-go-d93507931185
// https://coderwall.com/p/wohavg/creating-a-simple-tcp-server-in-go
// https://stackoverflow.com/questions/8309648/netcat-streaming-using-udp
// https://stackoverflow.com/questions/47231085/go-stdout-stream-from-async-command
// https://gowebexamples.com/websockets/

// scionlab chat
// netcat -local 1-ffaa:1:3b,[127.0.0.1] 1-ffaa:1:120,[127.0.0.1] 4141
// netcat -l -local 1-ffaa:1:120,[127.0.0.1] 4141

// test chat
// netcat -local 1-ff00:0:111,[127.0.0.1] 1-ff00:0:112,[127.0.0.2] 4141
// netcat -l -local 1-ff00:0:112,[127.0.0.2] 4141

// test movie
// cat ~/Desktop/out-2019-08-06.ogv | netcat -b -local 1-ff00:0:112,[127.0.0.2] 1-ff00:0:111,[127.0.0.1] 4142
// netcat -l -b -local 1-ff00:0:111,[127.0.0.1] 4142 | mplayer -

var yourMsg, logMsgs, socket;

$(document).ready(function() {
    yourMsg = document.getElementById("yourMsg");
    logMsgs = document.getElementById("logMsgs");
});

function openNetcatChat(localAddr, remoteAddr) {
    socket = new WebSocket(encodeURI("ws://" + document.location.host
            + "/echo?local=" + localAddr + "&remote=" + remoteAddr));
    socket.onopen = function() {
        appendLog("Status: WS connected\n");
    };

    socket.onmessage = function(e) {
        appendLog("Friend: " + e.data + "\n");
    };
}

function sendMsg() {
    if (socket) {
        socket.send(yourMsg.value);
        appendLog("You: " + yourMsg.value + "\n");
        yourMsg.value = "";
    }
}

function appendLog(msg) {
    var doScroll = logMsgs.scrollTop > logMsgs.scrollHeight
            - logMsgs.clientHeight - 1;
    var item = document.createElement("div");
    item.innerHTML = msg;
    logMsgs.appendChild(item);
    if (doScroll) {
        logMsgs.scrollTop = logMsgs.scrollHeight - logMsgs.clientHeight;
    }
}
