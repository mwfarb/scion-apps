package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	//"github.com/mattn/go-xmpp"
	//"gosrc.io/xmpp"
	//"gosrc.io/xmpp/stanza"

	"github.com/gorilla/websocket"
	log "github.com/inconshreveable/log15"
	"github.com/kormat/fmt15"
	lib "github.com/netsec-ethz/scion-apps/webapp/lib"
	. "github.com/netsec-ethz/scion-apps/webapp/util"
)

var id = "chat"
var templates *template.Template
var myIA = "1-ff00:0:111" // TODO: remove debug

// Page holds default fields for html template expansion for each page.
type Page struct {
	Title string
	MyIA  string
}

var options lib.CmdOptions

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

	// prepare templates
	templates = prepareTemplates(options.StaticRoot)
	log.Info("IA loaded:", "myIa", myIA)

	checkPath(options.AppsRoot)
	appsBuildCheck("scion-netcat")

	//initXmpp(myIA, "scion-xmpp.cylab.cmu.edu")

	initServeHandlers()
	log.Info(fmt.Sprintf("Browser access: at http://%s:%d.", options.Addr, options.Port))
	log.Info(fmt.Sprintf("Listening on %s:%d...", options.Addr, options.Port))
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", options.Addr, options.Port), logRequestHandler(http.DefaultServeMux))
	CheckFatal(err)
}

func initServeHandlers() {
	serveExact("/favicon.ico", path.Join(options.StaticRoot, "favicon.ico"))
	fsStatic := http.FileServer(http.Dir(path.Join(options.StaticRoot, "static")))
	http.Handle("/static/", http.StripPrefix("/static/", fsStatic))
	http.HandleFunc("/", chatHandler)
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

func chatVideoHandler(w http.ResponseWriter, r *http.Request) {
	local := r.FormValue("local")
	remote := r.FormValue("remote")

	localAddr := local[:strings.LastIndex(local, ":")]
	localPort := local[strings.LastIndex(local, ":")+1:]
	remoteAddr := remote[:strings.LastIndex(remote, ":")]
	remotePort := remote[strings.LastIndex(remote, ":")+1:]

	// open websocket server connection
	conn, err := upgrader.Upgrade(w, r, nil)
	defer conn.Close()
	if CheckError(err) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	installpath := getClientLocationBin("scion-netcat")
	cmdloc := fmt.Sprintf("-local=%s", localAddr)
	keyPath := path.Join(options.StaticRoot, "key.pem")
	certPath := path.Join(options.StaticRoot, "cert.pem")
	cmdKey := fmt.Sprintf("-tlsKey=%s", keyPath)
	cmdCert := fmt.Sprintf("-tlsCert=%s", certPath)

	// serve
	serveArgs := []string{installpath, cmdloc, remoteAddr, remotePort}
	log.Info("Run NC Video:", "command", strings.Join(serveArgs, " "))
	commandServe := exec.Command(serveArgs[0], serveArgs[1:]...)
	// open scion netcat serve to friend and ready stdin...
	stdin, err := commandServe.StdinPipe()
	CheckError(err)
	err = commandServe.Start()
	CheckError(err)
	// monitor websocket for input
	go func() {
		for {
			// message from browser
			_, buf, err := conn.ReadMessage()
			CheckError(err)
			// pipe buf to netcat stdin...
			log.Debug("netcat v send:", "buflen", len(buf))
			stdin.Write(buf)
		}
	}()
	err = commandServe.Wait()
	CheckError(err)

	// listen
	listenArgs := []string{installpath, "-l", cmdloc, localPort, cmdKey, cmdCert}
	log.Info("Run NC Video:", "command", strings.Join(listenArgs, " "))
	commandListen := exec.Command(listenArgs[0], listenArgs[1:]...)
	// open scion netcat client listen from friend and ready stdout...
	stdout, err := commandListen.StdoutPipe()
	CheckError(err)
	reader := bufio.NewReader(stdout)
	buf := make([]byte, 1024)
	go func(reader io.Reader) {
		for {
			n, err := reader.Read(buf)
			fmt.Println(n, err, buf[:n])
			if err == io.EOF {
				break
			}
			// received buf on stdin while running netcat...
			log.Debug("netcat v recv:", "buflen", len(buf))
			// send buf to browser
			err = conn.WriteMessage(websocket.BinaryMessage, buf)
			CheckError(err)
		}
	}(reader)
	err = commandListen.Start()
	CheckError(err)
	err = commandListen.Wait()
	CheckError(err)
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

	localAddr := local[:strings.LastIndex(local, ":")]
	localPort := local[strings.LastIndex(local, ":")+1:]
	remoteAddr := remote[:strings.LastIndex(remote, ":")]
	remotePort := remote[strings.LastIndex(remote, ":")+1:]

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
	cmdloc := fmt.Sprintf("-local=%s", localAddr)
	keyPath := path.Join(options.StaticRoot, "key.pem")
	certPath := path.Join(options.StaticRoot, "cert.pem")
	cmdKey := fmt.Sprintf("-tlsKey=%s", keyPath)
	cmdCert := fmt.Sprintf("-tlsCert=%s", certPath)

	// TODO: (mwfarb) add reasonable retry logic when handshake timeout occurs

	// serve
	serveArgs := []string{installpath, cmdloc, remoteAddr, remotePort}
	log.Info("Executing:", "command", strings.Join(serveArgs, " "))
	commandServe := exec.Command(serveArgs[0], serveArgs[1:]...)
	// open scion netcat serve to friend and ready stdin...
	stdin, err := commandServe.StdinPipe()
	CheckError(err)
	err = commandServe.Start()
	CheckError(err)
	// monitor websocket for input
	go func() {
		for {
			// message from browser
			_, msg, err := conn.ReadMessage()
			CheckError(err)
			// pipe message to netcat stdin...
			log.Debug("netcat t send:", "msg", string(msg))
			stdin.Write(append(msg, '\n'))
		}
	}()
	err = commandServe.Wait()
	CheckError(err)

	// listen
	listenArgs := []string{installpath, "-l", cmdloc, localPort, cmdKey, cmdCert}
	log.Info("Executing:", "command", strings.Join(listenArgs, " "))
	commandListen := exec.Command(listenArgs[0], listenArgs[1:]...)
	// open scion netcat client listen from friend and ready stdout...
	stdout, err := commandListen.StdoutPipe()
	CheckError(err)
	reader := bufio.NewReader(stdout)
	go func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			// received text on stdin while running netcat...
			msg := scanner.Text()
			log.Debug("netcat t recv:", "msg", string(msg))
			// send message to browser
			err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
			CheckError(err)
		}
	}(reader)
	err = commandListen.Start()
	CheckError(err)
	err = commandListen.Wait()
	CheckError(err)
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
