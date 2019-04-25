package main

import (
	golog "log"
	"os"
	"strconv"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/netsec-ethz/scion-apps/ssh/config"
	"github.com/netsec-ethz/scion-apps/ssh/quicconn"
	"github.com/netsec-ethz/scion-apps/ssh/scionutils"
	"github.com/netsec-ethz/scion-apps/ssh/server/serverconfig"
	"github.com/netsec-ethz/scion-apps/ssh/server/ssh"
	"github.com/netsec-ethz/scion-apps/ssh/utils"

	log "github.com/inconshreveable/log15"
)

const (
	version = "1.0"
)

var (
	// Connection
	listenAddress   = kingpin.Flag("address", "SCION address to listen on").Default("").String()
	useIASCIoNDPath = kingpin.Flag("sciond-path-from-ia", "Use IA address when resolving SCIOND socket path").Short('P').Bool()
	options         = kingpin.Flag("option", "Set an option").Short('o').Strings()

	// Configuration file
	configurationFile = kingpin.Flag("config-file", "SSH server configuration file").Short('f').Default("/etc/ssh/sshd_config").ExistingFile()
)

func setConfIfNot(conf *serverconfig.ServerConfig, name string, value, not interface{}) bool {
	res, err := config.SetIfNot(conf, name, value, not)
	if err != nil {
		golog.Panicf("Error setting option %s to %v: %v", name, value, err)
	}
	return res
}

func createConfig() *serverconfig.ServerConfig {
	conf := serverconfig.Create()

	updateConfigFromFile(conf, *configurationFile)

	for _, option := range *options {
		err := config.UpdateFromString(conf, option)
		if err != nil {
			log.Debug("Error updating config from --option flag: %v", err)
		}
	}

	// TODO: Set port from listening address
	// setConfIfNot(conf, "Port", *PORT, 0)

	return conf
}

func updateConfigFromFile(conf *serverconfig.ServerConfig, pth string) {
	err := config.UpdateFromFile(conf, utils.ParsePath(pth))
	if err != nil {
		if !os.IsNotExist(err) {
			golog.Panicf("Error updating config from file %s: %v", pth, err)
		}
	}
}

func main() {
	kingpin.Parse()

	log.Debug("Starting SCION SSH server...")

	conf := createConfig()

	err := scionutils.InitSCIONConnection(utils.ParsePath(conf.QUICKeyPath), utils.ParsePath(conf.QUICCertificatePath), *useIASCIoNDPath)
	if err != nil {
		golog.Panicf("Error initializing SCION connection: %v", err)
	}

	sshServer, err := ssh.Create(conf, version)
	if err != nil {
		golog.Panicf("Error creating ssh server: %v", err)
	}

	port, err := strconv.Atoi(conf.Port)
	if err != nil {
		golog.Panicf("Can't parse port %v: %v", conf.Port, err)
	}

	log.Debug("Currently, ListenAddress.Port is ignored (only value from config taken)")
	listener, err := scionutils.ListenSCION(uint16(port))
	if err != nil {
		golog.Panicf("Failed to listen (%v)", err)
	}

	log.Debug("Starting to wait for connections")
	for {
		//TODO: Check when to close the connections
		sess, err := listener.Accept()
		if err != nil {
			log.Debug("Failed to accept session: %v", err)
			continue
		}
		stream, err := sess.AcceptStream()
		if err != nil {
			log.Debug("Failed to accept incoming connection (%v)", err)
			continue
		}

		qc := &quicconn.QuicConn{Session: sess, Stream: stream}
		go sshServer.HandleConnection(qc)
	}
}
