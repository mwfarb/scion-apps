// signaling server
var firebaseConfig = {
    authDomain : "wip-scion-webrtc.firebaseapp.com",
    databaseURL : "https://wip-scion-webrtc.firebaseio.com",
    projectId : "wip-scion-webrtc",
    storageBucket : "wip-scion-webrtc.appspot.com",
};

var iceServers = {
    'iceServers' : [ {
        'urls' : 'stun:stun.services.mozilla.com'
    }, {
        'urls' : 'stun:stun.l.google.com:19302'
    } ]
};

const regexIpAddr = /[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}/;

var pc = new RTCPeerConnection(iceServers);
var database;

var yourVideo;
var friendsVideo;

var yourId = Math.floor(Math.random() * 1000000000);
var yourIa;
var yourAddr;
var yourChatPort = randomPort();
var yourAudioPort = randomPort();
var yourVideoPort = randomPort();

var friendsIa;
var friendsAddr;
var friendsChatPort;
var friendsAudioPort;
var friendsVideoPort;

window.onload = function(event) {
    $("#videoout").empty();
    debugLog("Loaded page");
    yourVideo = document.getElementById("yourVideo");
    friendsVideo = document.getElementById("friendsVideo");
    showMyIa(yourIa);
    debugLog("yourId: " + yourId);
    debugLog("yourChatPort: " + yourChatPort);
    debugLog("yourAudioPort: " + yourAudioPort);
    debugLog("yourVideoPort: " + yourVideoPort);
    ajaxVizConfig();
    ajaxAsTopo();
    ajaxChatConfig();
};

function debugLog(msg) {
    console.log(msg);
    $("#videoout").append("\n" + msg + ".");
    $(".stdout").scrollTop($(".stdout")[0].scrollHeight);
}

function showError(err) {
    if (err && err != '') {
        console.error(err);
        $("#as-error").html(err);
        $("#as-error").css('color', 'red');
    } else {
        $("#as-error").empty();
    }
}

function reduceHex(hex) {
    return parseInt(hex, 16).toString(16);
}

function iaRawCompose(rawIa) {
    var parts = [];
    var hex = rawIa.toString(16);
    var isd = parseInt(hex.slice(0, -12), 16);
    var as1 = reduceHex(hex.slice(-12, -8));
    var as2 = reduceHex(hex.slice(-8, -4));
    var as3 = reduceHex(hex.slice(-4));
    return [ isd, as1, as2, as3 ];
}

function iaRaw2Read(rawIa) {
    var parts = iaRawCompose(rawIa);
    return parts[0] + '-' + parts.slice(1, 4).join(':');
}

function iaRaw2File(rawIa) {
    var parts = iaRawCompose(rawIa);
    return parts[0] + '-' + parts.slice(1, 4).join('_');
}

function ipv4Raw2Read(rawIpv4) {
    var b = atob(rawIpv4); // decode
    var a = new Uint8Array(str2ab(b));
    var ipv4 = a.join('.');
    return ipv4;
}

function ajaxChatConfig() {
    return $.ajax({
        url : 'chatcfg',
        type : 'post',
        timeout : 10000,
        success : isChatConfigComplete,
        error : function(jqXHR, textStatus, errorThrown) {
            showError(this.url + ' ' + textStatus + ': ' + errorThrown);
        },
    });
}

function isChatConfigComplete(data, textStatus, jqXHR) {
    console.debug(data);
}

function ajaxAsTopo() {
    return $.ajax({
        url : 'getastopo',
        type : 'post',
        dataType : "json",
        data : {
            "src" : yourIa
        },
        timeout : 10000,
        success : isAsTopoComplete,
        error : function(jqXHR, textStatus, errorThrown) {
            showError(this.url + ' ' + textStatus + ': ' + errorThrown);
        },
    });
}

function isAsTopoComplete(data, textStatus, jqXHR) {
    console.debug(data);
    yourIa = iaRaw2Read(data.as_info.Entries[0].RawIsdas);
    showMyIa(yourIa);
    yourAddr = data.if_info["1"].IP;
    yourPortIf = data.if_info["1"].Port;
    debugLog("yourAddr: " + yourAddr);
    debugLog("yourPortIf: " + yourPortIf);
}

function ajaxVizConfig() {
    return $.ajax({
        url : 'config',
        type : 'get',
        dataType : "json",
        timeout : 30000,
        success : isRTCConfigComplete,
        error : function(jqXHR, textStatus, errorThrown) {
            debugLog(this.url + ' ' + textStatus + ': ' + errorThrown);
        },
    });
}

function isRTCConfigComplete(data, textStatus, jqXHR) {
    debugLog(this.url + ' ' + textStatus);
    debugLog('firebaseConfig.apiKey = ' + data.webrtc_apiKey);
    debugLog('firebaseConfig.messagingSenderId = '
            + data.webrtc_messagingSenderId);
    debugLog('firebaseConfig.appId = ' + data.webrtc_appId);
    firebaseConfig.apiKey = data.webrtc_apiKey;
    firebaseConfig.messagingSenderId = data.webrtc_messagingSenderId;
    firebaseConfig.appId = data.webrtc_appId;
    // Initialize Firebase
    firebase.initializeApp(firebaseConfig);
    database = firebase.database().ref();
    pc.onicecandidate = function(event) {
        if (event.candidate) {
            sendMessage(yourId, JSON.stringify({
                'ice' : event.candidate
            }));
        } else {
            console.log("Sent All Ice");

            // addresses now complete?, so open netcat chat

            // netcat listen to stdout on local IA yourChatPort
            // - on stdout.read(msg), append.txt("friend:"+msg)

            // netcat serve to stdin on remote IA friendChatPort
            // - on btn-send(msg), stdin.write(), append.txt("self:"+msg)
            if (friendsIa && friendsAddr && friendsChatPort) {
                var local = formatScionAddr(yourIa, yourAddr, yourChatPort);
                var remote = formatScionAddr(friendsIa, friendsAddr,
                        friendsChatPort);
                openNetcatChatText(local, remote);
                openNetcatChatVideo(local, remote);
            }
        }
    };
    pc.onaddstream = function(event) {
        friendsVideo.srcObject = event.stream;
        // setupSigConnection();
        // $("#hangup-button").disabled = false;
    };
    database.on('child_added', readMessage);
    // when load finished...
    showMyFace();
}

function formatScionAddr(ia, addr, port) {
    return ia + ",[" + addr + "]:" + port;
}

function randomPort() {
    return Math.floor(Math.random() * 10000) + 30000;
}

function sendMessage(senderId, data) {
    debugLog("Called sendMessage()");
    var msg = database.push({
        sender : senderId,
        ia : yourIa,
        addr : yourAddr,
        portC : yourChatPort,
        portA : yourAudioPort,
        portV : yourVideoPort,
        message : data
    });
    msg.remove();
    updateAddrs();
}

function readMessage(data) {
    debugLog("Called readMessage()");
    var msg = JSON.parse(data.val().message);
    var sender = data.val().sender;
    if (sender != yourId) {
        friendsIa = data.val().ia;
        friendsAddr = data.val().addr;
        friendsChatPort = data.val().portC;
        friendsAudioPort = data.val().portA;
        friendsVideoPort = data.val().portV;
        showPeerIa(friendsIa);
        if (msg.ice != undefined) {
            pc.addIceCandidate(new RTCIceCandidate(msg.ice));
        } else if (msg.sdp) {
            if (msg.sdp.type == "offer") {
                // received offer, store as remote conn
                pc.setRemoteDescription(new RTCSessionDescription(msg.sdp))
                // create answer
                .then(function() {
                    return pc.createAnswer();
                })
                // store answer as local conn
                .then(function(answer) {
                    pc.setLocalDescription(answer);
                })
                // send answer
                .then(function() {
                    sendMessage(yourId, JSON.stringify({
                        'sdp' : pc.localDescription
                    }));
                });
            } else if (msg.sdp.type == "answer") {
                pc.setRemoteDescription(new RTCSessionDescription(msg.sdp));
            }
        }
    }
    updateAddrs();
};

function updateAddrs() {
    if (pc.currentLocalDescription) {
        debugLog("yourType: " + pc.currentLocalDescription.type);
        showMyAddr(getSdpAddr(pc.currentLocalDescription.sdp));
        showMyAudio(getSdpAudio(pc.currentLocalDescription.sdp));
        showMyVideo(getSdpVideo(pc.currentLocalDescription.sdp));
    }
    if (pc.currentRemoteDescription) {
        debugLog("friendsType: " + pc.currentRemoteDescription.type);
        showPeerAddr(getSdpAddr(pc.currentRemoteDescription.sdp));
        showPeerAudio(getSdpAudio(pc.currentRemoteDescription.sdp));
        showPeerVideo(getSdpVideo(pc.currentRemoteDescription.sdp));
    }
}

function showMyFace() {
    debugLog("Called showMyFace()");
    navigator.mediaDevices.getUserMedia({
        audio : true,
        video : true
    })
    // place your media in local object
    .then(function(stream) {
        yourVideo.srcObject = stream;
        // sendVideo(stream);
        return stream;
    })
    // add your media to stream
    .then(function(stream) {
        pc.addStream(stream);
    });
}

function showFriendsFace() {
    debugLog("Called showFriendsFace()");

    pc.createOffer()
    // place offer in local conn
    .then(function(offer) {
        pc.setLocalDescription(offer);
    })
    // send the offer
    .then(function() {
        sendMessage(yourId, JSON.stringify({
            'sdp' : pc.localDescription
        }));
    });
}

function showMyIa(ia) {
    debugLog("yourIaText: " + ia);
    $("#yourIaText").html(ia);
}

function showPeerIa(ia) {
    debugLog("friendsIaText: " + ia);
    $("#friendsIaText").html(ia);
}

function showMyAddr(addr) {
    debugLog("yourVar1: " + addr);
    $("#yourVar1").html(addr);
}

function showPeerAddr(addr) {
    debugLog("friendsVar1: " + addr);
    $("#friendsVar1").html(addr);
}

function showMyAudio(addr) {
    debugLog("yourVar2: " + addr);
    $("#yourVar2").html(addr);
}

function showPeerAudio(addr) {
    debugLog("friendsVar2: " + addr);
    $("#friendsVar2").html(addr);
}

function showMyVideo(addr) {
    debugLog("yourVar3: " + addr);
    $("#yourVar3").html(addr);
}

function showPeerVideo(addr) {
    debugLog("friendsVar3: " + addr);
    $("#friendsVar3").html(addr);
}

function getSdpAddr(sdp) {
    var ips = [];
    if (!sdp) {
        return null;
    }
    ips = sdp.split('\r\n').filter(function(line) {
        return line.indexOf('c=') === 0;
    }).map(function(ipstr) {
        return ipstr.match(regexIpAddr)[0];
    });
    return ips[0];
}

function getSdpAudio(sdp) {
    var ips = [];
    if (sdp) {
        ips = sdp.split('\r\n').filter(function(line) {
            return line.indexOf('m=audio') === 0;
        });
    }
    return ips[0];
}

function getSdpVideo(sdp) {
    var ips = [];
    if (sdp) {
        ips = sdp.split('\r\n').filter(function(line) {
            return line.indexOf('m=video') === 0;
        });
    }
    return ips[0];
}
