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

var yourMsg, yourVideo, logMsgs, socketT, socketV;

$(document).ready(function() {
    yourMsg = document.getElementById("yourMsg");
    yourVideo = document.getElementById("yourVideo");
    logMsgs = document.getElementById("logMsgs");
});

function openNetcatChatText(localAddr, remoteAddr) {

    // TODO chrome throws an exception

    socketT = new WebSocket(encodeURI("ws://" + document.location.host
            + "/wschat?local=" + localAddr + "&remote=" + remoteAddr));

    socketT.onopen = function() {
        appendChatDisplay("Status: WS connected\n");
    };

    socketT.onmessage = function(e) {
        appendChatDisplay("Friend: " + e.data + "\n");
    };
}

function openNetcatChatVideo(localAddr, remoteAddr) {
    socketV = new WebSocket(encodeURI("ws://" + document.location.host
            + "/wsvideo?local=" + localAddr + "&remote=" + remoteAddr));

    socketV.onopen = function() {
        // TODO appendChatDisplay("Status: WS connected\n");
    };

    socketV.onmessage = function(e) {
        // TODO appendChatDisplay("Friend: " + e.data + "\n");
    };
}

function sendTextMsg() {
    if (socketT) {
        socketT.send(yourMsg.value);
        appendChatDisplay("You: " + yourMsg.value + "\n");
        yourMsg.value = "";
    }
}

function sendVideoStream() {
    if (socketV) {
        socketT.send(yourMsg.value);
        appendChatDisplay("You: " + yourMsg.value + "\n");
        yourMsg.value = "";
    }
}

function appendChatDisplay(msg) {
    var doScroll = logMsgs.scrollTop > logMsgs.scrollHeight
            - logMsgs.clientHeight - 1;
    var item = document.createElement("div");
    item.innerHTML = msg;
    logMsgs.appendChild(item);
    if (doScroll) {
        logMsgs.scrollTop = logMsgs.scrollHeight - logMsgs.clientHeight;
    }
}
