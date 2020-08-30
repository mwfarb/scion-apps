package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	//"github.com/mattn/go-xmpp"
	//"gosrc.io/xmpp"
	//"gosrc.io/xmpp/stanza"

	"github.com/gorilla/websocket"
	log "github.com/inconshreveable/log15"
	"github.com/kormat/fmt15"
	lib "github.com/netsec-ethz/scion-apps/chat/lib"
	. "github.com/netsec-ethz/scion-apps/chat/util"
	"github.com/scionproto/scion/go/lib/addr"
	"github.com/scionproto/scion/go/lib/common"
	"github.com/scionproto/scion/go/lib/sciond"
	"github.com/scionproto/scion/go/proto"
)

var id = "chat"
var templates *template.Template
var myIA string
var options lib.CmdOptions

// Configuations to save. Zeroing out any of these placeholders will cause the
// webserver to request a fresh external copy to keep locally.
var cConfig string

// Page holds default fields for html template expansion for each page.
type Page struct {
	Title string
	MyIA  string
}

func checkPath(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		CheckError(err)
	}
}

func main() {
	options = lib.ParseFlags()

	// logging
	log.Root().SetHandler(log.MultiHandler(
		log.LvlFilterHandler(log.LvlDebug,
			log.StreamHandler(os.Stderr, fmt15.Fmt15Format(fmt15.ColorMap))),
		log.LvlFilterHandler(log.LvlInfo,
			log.Must.FileHandler(path.Join(options.StaticRoot, fmt.Sprintf("%s.log", id)),
				fmt15.Fmt15Format(nil)))))

	// load IA from default sciond or passed in sciond address
	c, err := connect(options.Sciond)
	if CheckError(err) {
		return
	}
	asir, err := c.ASInfo(context.Background(), addr.IA{})
	if CheckError(err) {
		return
	}
	myIA = asir.Entries[0].RawIsdas.String()

	// prepare templates
	templates = prepareTemplates(options.StaticRoot)
	log.Info("IA loaded:", "myIa", myIA)

	checkPath(options.AppsRoot)
	appsBuildCheck("scion-netcat")

	//initXmpp(myIA, "scion-xmpp.cylab.cmu.edu")

	initServeHandlers()
	log.Info(fmt.Sprintf("Browser access: at http://%s:%d.", options.Addr, options.Port))
	log.Info(fmt.Sprintf("Listening on %s:%d...", options.Addr, options.Port))
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", options.Addr, options.Port), logRequestHandler(http.DefaultServeMux))
	CheckFatal(err)
}

func initServeHandlers() {
	serveExact("/favicon.ico", path.Join(options.StaticRoot, "favicon.ico"))
	fsStatic := http.FileServer(http.Dir(path.Join(options.StaticRoot, "static")))
	http.Handle("/static/", http.StripPrefix("/static/", fsStatic))
	http.HandleFunc("/", chatHandler)
	http.HandleFunc("/config", ConfigHandler)
	http.HandleFunc("/getastopo", AsTopoHandler)
	http.HandleFunc("/wschat", chatTextHandler)
	http.HandleFunc("/wsvideo", chatVideoHandler)
	http.HandleFunc("/chatcfg", chatConfigHandler)
}

func logRequestHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info(fmt.Sprintf("%s %s %s", r.RemoteAddr, r.Method, r.URL))
		handler.ServeHTTP(w, r)
	})
}

func prepareTemplates(srcpath string) *template.Template {
	return template.Must(template.ParseFiles(
		path.Join(srcpath, "template/chat.html"),
	))
}

func serveExact(pattern string, filename string) {
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filename)
	})
}

func display(w http.ResponseWriter, tmpl string, data interface{}) {
	templates.ExecuteTemplate(w, tmpl, data)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	display(w, "chat", &Page{Title: "SCIONLab Chat", MyIA: myIA})
}

func appsBuildCheck(app string) {
	installpath := getClientLocationBin(app)
	if _, err := os.Stat(installpath); os.IsNotExist(err) {
		CheckError(err)
		CheckError(errors.New("App missing, build all apps with 'make install'"))
	} else {
		log.Info(fmt.Sprintf("Existing install, found %s...", app))
	}
}

// Parses html selection and returns name of app binary.
func getClientLocationBin(app string) string {
	var binname string
	switch app {
	case "scion-netcat":
		binname = path.Join(options.AppsRoot, "scion-netcat")
	}
	return binname
}

func returnError(w http.ResponseWriter, err error) {
	fmt.Fprint(w, `{"err":`+strconv.Quote(err.Error())+`}`)
}

// connect opens a connection to the scion daemon at sciondAddress or, if
// empty, the default address.
func connect(sciondAddress string) (sciond.Connector, error) {
	if len(sciondAddress) == 0 {
		sciondAddress = sciond.DefaultSCIONDAddress
	}
	sciondConn, err := sciond.NewService(sciondAddress).Connect(context.Background())
	if CheckError(err) {
		return nil, err
	}
	return sciondConn, nil
}

// AsTopoHandler handles requests for AS data, returning results from sciond.
func AsTopoHandler(w http.ResponseWriter, r *http.Request) {
	c, err := connect(options.Sciond)
	if CheckError(err) {
		returnError(w, err)
		return
	}

	asir, err := c.ASInfo(context.Background(), addr.IA{})
	if CheckError(err) {
		returnError(w, err)
		return
	}
	ajsonInfo, _ := json.Marshal(asir)
	log.Debug("AsTopoHandler:", "ajsonInfo", string(ajsonInfo))

	ifirs, err := c.IFInfo(context.Background(), []common.IFIDType{})
	if CheckError(err) {
		returnError(w, err)
		return
	}
	ijsonInfo, _ := json.Marshal(ifirs)
	log.Debug("AsTopoHandler:", "ijsonInfo", string(ijsonInfo))

	svcirs, err := c.SVCInfo(context.Background(), []proto.ServiceType{
		proto.ServiceType_bs, proto.ServiceType_ps, proto.ServiceType_cs,
		proto.ServiceType_sb, proto.ServiceType_sig, proto.ServiceType_ds})
	if CheckError(err) {
		returnError(w, err)
		return
	}
	sjsonInfo, _ := json.Marshal(svcirs)
	log.Debug("AsTopoHandler:", "sjsonInfo", string(sjsonInfo))

	fmt.Fprintf(w, fmt.Sprintf(`{"as_info":%s,"if_info":%s,"svc_info":%s}`,
		ajsonInfo, ijsonInfo, sjsonInfo))
}

// ConfigHandler handles requests for configurable, centralized data sources.
func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	projectID := "my-project-1470640410708"
	url := fmt.Sprintf("https://%s.appspot.com/getconfig", projectID)
	if len(cConfig) == 0 {
		buf := new(bytes.Buffer)
		resp, err := http.Post(url, "application/json", buf)
		if CheckError(err) {
			returnError(w, err)
			return
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		cConfig = string(body)
		log.Debug("ConfigHandler:", "cached", cConfig)
	}
	fmt.Fprint(w, cConfig)
}

func chatVideoHandler(w http.ResponseWriter, r *http.Request) {
	// 	local := r.FormValue("local")
	// 	remote := r.FormValue("remote")

	// 	//localAddr := local[:strings.LastIndex(local, ":")]
	// 	localPort := local[strings.LastIndex(local, ":")+1:]
	// 	//remoteAddr := remote[:strings.LastIndex(remote, ":")]
	// 	//remotePort := remote[strings.LastIndex(remote, ":")+1:]

	// 	// open websocket server connection
	// 	conn, err := upgrader.Upgrade(w, r, nil)
	// 	defer conn.Close()
	// 	if CheckError(err) {
	// 		http.Error(w, err.Error(), http.StatusInternalServerError)
	// 		return
	// 	}

	// 	installpath := getClientLocationBin("scion-netcat")
	// 	//cmdloc := fmt.Sprintf("-local=%s", localAddr)
	// 	//keyPath := path.Join(options.StaticRoot, "key.pem")
	// 	//certPath := path.Join(options.StaticRoot, "cert.pem")
	// 	//cmdKey := fmt.Sprintf("-tlsKey=%s", keyPath)
	// 	//cmdCert := fmt.Sprintf("-tlsCert=%s", certPath)

	// 	// serve
	// 	//serveArgs := []string{installpath, cmdloc, remoteAddr, remotePort, "-b"}
	// 	serveArgs := []string{installpath, remote, "-b"}
	// 	log.Info("Run NC Video:", "command", strings.Join(serveArgs, " "))
	// 	commandServe := exec.Command(serveArgs[0], serveArgs[1:]...)
	// 	// open scion netcat serve to friend and ready stdin...
	// 	stdin, err := commandServe.StdinPipe()
	// 	CheckError(err)
	// 	err = commandServe.Start()
	// 	CheckError(err)
	// 	// monitor websocket for input
	// 	go func() {
	// 		for {
	// 			// message from browser
	// 			_, buf, err := conn.ReadMessage()
	// 			CheckError(err)
	// 			// pipe buf to netcat stdin...
	// 			log.Debug("netcat v send:", "buflen", len(buf))
	// 			stdin.Write(buf)
	// 		}
	// 	}()
	// 	err = commandServe.Wait()
	// 	CheckError(err)

	// 	// listen
	// 	//listenArgs := []string{installpath, "-l", cmdloc, localPort, cmdKey, cmdCert, "-b"}
	// 	listenArgs := []string{installpath, "-l", localPort, "-b"}
	// 	log.Info("Run NC Video:", "command", strings.Join(listenArgs, " "))
	// 	commandListen := exec.Command(listenArgs[0], listenArgs[1:]...)
	// 	// open scion netcat client listen from friend and ready stdout...
	// 	stdout, err := commandListen.StdoutPipe()
	// 	CheckError(err)
	// 	reader := bufio.NewReader(stdout)
	// 	buf := make([]byte, 1024)
	// 	go func(reader io.Reader) {
	// 		for {
	// 			n, err := reader.Read(buf)
	// 			fmt.Println(n, err, buf[:n])
	// 			if err == io.EOF {
	// 				break
	// 			}
	// 			// received buf on stdin while running netcat...
	// 			log.Debug("netcat v recv:", "buflen", len(buf))
	// 			// send buf to browser
	// 			err = conn.WriteMessage(websocket.BinaryMessage, buf)
	// 			CheckError(err)
	// 		}
	// 	}(reader)
	// 	err = commandListen.Start()
	// 	CheckError(err)
	// 	err = commandListen.Wait()
	// 	CheckError(err)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO fix, ensure that
	},
}

func chatConfigHandler(w http.ResponseWriter, r *http.Request) {
	// find TLS cert, or generate if missing
	// openssl req -newkey rsa:2048 -nodes -keyout ./key.pem -x509 -days 365 -out ./cert.pem -subj '/CN=localhost'
	keyPath := path.Join(options.StaticRoot, "key.pem")
	certPath := path.Join(options.StaticRoot, "cert.pem")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		certArgs := []string{"openssl", "req",
			"-newkey", "rsa:2048",
			"-nodes",
			"-keyout", keyPath,
			"-x509",
			"-days", "365",
			"-out", certPath,
			"-subj", "/CN=localhost"}
		log.Info("Executing:", "command", strings.Join(certArgs, " "))
		cmd := exec.Command(certArgs[0], certArgs[1:]...)

		var outb, errb bytes.Buffer
		cmd.Stdout = &outb
		cmd.Stderr = &errb
		err := cmd.Run()
		CheckError(err)
		log.Info("results:", "out:", outb.String(), "err:", errb.String())
	}
}

func chatTextHandler(w http.ResponseWriter, r *http.Request) {
	local := r.FormValue("local")
	remote := r.FormValue("remote")

	//localAddr := local[:strings.LastIndex(local, ":")]
	localPort := local[strings.LastIndex(local, ":")+1:]
	//remoteAddr := remote[:strings.LastIndex(remote, ":")]
	//remotePort := remote[strings.LastIndex(remote, ":")+1:]

	// open websocket server connection
	conn, err := upgrader.Upgrade(w, r, nil)
	defer conn.Close()
	if CheckError(err) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// use passed in ports for servers/clients here
	// scion-netcat -local 1-ff00:0:111,[127.0.0.1] 1-ff00:0:112,[127.0.0.2] 4141
	// scion-netcat -l -local 1-ff00:0:112,[127.0.0.2] 4141
	installpath := getClientLocationBin("scion-netcat")
	//cmdloc := fmt.Sprintf("-local=%s", localAddr)
	//keyPath := path.Join(options.StaticRoot, "key.pem")
	//certPath := path.Join(options.StaticRoot, "cert.pem")
	//cmdKey := fmt.Sprintf("-tlsKey=%s", keyPath)
	//cmdCert := fmt.Sprintf("-tlsCert=%s", certPath)

	// TODO: (mwfarb) add reasonable retry logic when handshake timeout occurs

	// serve
	//serveArgs := []string{installpath, cmdloc, remoteAddr, remotePort, "-b"}
	serveArgs := []string{installpath, "-b", remote}
	log.Info("Executing:", "command", strings.Join(serveArgs, " "))
	commandServe := exec.Command(serveArgs[0], serveArgs[1:]...)
	// open scion netcat serve to friend and ready stdin...
	stdin, err := commandServe.StdinPipe()
	if CheckError(err) {
		returnError(w, err)
		return
	}
	err = commandServe.Start()
	if CheckError(err) {
		returnError(w, err)
		return
	}
	// monitor websocket for input
	go func() {
		for {
			// message from browser
			_, msg, err := conn.ReadMessage()
			if CheckError(err) {
				returnError(w, err)
				return
			}
			// pipe message to netcat stdin...
			log.Debug("netcat t send:", "msg", string(msg))
			stdin.Write(append(msg, '\n'))
		}
	}()
	err = commandServe.Wait()
	if CheckError(err) {
		returnError(w, err)
		return
	}

	// listen
	//listenArgs := []string{installpath, "-l", cmdloc, localPort, cmdKey, cmdCert, "-b"}
	listenArgs := []string{installpath, "-b", "-l", localPort}
	log.Info("Executing:", "command", strings.Join(listenArgs, " "))
	commandListen := exec.Command(listenArgs[0], listenArgs[1:]...)
	// open scion netcat client listen from friend and ready stdout...
	stdout, err := commandListen.StdoutPipe()
	if CheckError(err) {
		returnError(w, err)
		return
	}
	reader := bufio.NewReader(stdout)
	go func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			// received text on stdin while running netcat...
			msg := scanner.Text()
			log.Debug("netcat t recv:", "msg", string(msg))
			// send message to browser
			err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if CheckError(err) {
				returnError(w, err)
				return
			}
		}
	}(reader)
	err = commandListen.Start()
	if CheckError(err) {
		returnError(w, err)
		return
	}
	err = commandListen.Wait()
	if CheckError(err) {
		returnError(w, err)
		return
	}
}

//func initXmpp(fileIa, host string) {
//     config := xmpp.Config{
//             TransportConfiguration: xmpp.TransportConfiguration{
//                     Address: host + ":5222",
//             },
//             Jid:          fileIa + "@" + host,
//             Credential:   xmpp.Password(fileIa),
//             StreamLogger: os.Stdout,
//             Insecure:     true,
//             // TLSConfig: tls.Config{InsecureSkipVerify: true},
//     }
//
//     router := xmpp.NewRouter()
//     router.HandleFunc("message", handleMessageXmpp)
//
//     client, err := xmpp.NewClient(&config, router, errorHandlerXmpp)
//     if err != nil {
//             log.Error("%+v", err)
//     }
//
//     // If you pass the client to a connection manager, it will handle the reconnect policy
//     // for you automatically.
//     cm := xmpp.NewStreamManager(client, nil)
//     err = cm.Run()
//     CheckError(err)
//}
//
//func handleMessageXmpp(s xmpp.Sender, p stanza.Packet) {
//     msg, ok := p.(stanza.Message)
//     if !ok {
//             _, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", p)
//             return
//     }
//
//     _, _ = fmt.Fprintf(os.Stdout, "Body = %s - from = %s\n", msg.Body, msg.From)
//     reply := stanza.Message{Attrs: stanza.Attrs{To: msg.From}, Body: msg.Body}
//     _ = s.Send(reply)
//}
//
//func errorHandlerXmpp(err error) {
//     fmt.Println(err.Error())
//}
