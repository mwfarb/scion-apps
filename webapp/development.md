# Webapp Development Notes

Your mileage may vary.

# SCIONLab VM Test Development

Add alternate test forwarding port line in `Vagrantfile`:
```
  config.vm.network "forwarded_port", guest: 8080, host: 8080, protocol: "tcp"
```

Install Go 1.11:
```shell
cd ~
curl -O https://dl.google.com/go/go1.11.13.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.11.13.linux-amd64.tar.gz
```

Update Go Paths:
```shell
echo 'export GOPATH="$HOME/go"' >> ~/.profile
echo 'export PATH="$HOME/.local/bin:$GOPATH/bin:/usr/local/go/bin:$PATH"' >> ~/.profile
source ~/.profile
mkdir -p "$GOPATH"
```

Build `scion-apps`:
```shell
go get github.com/netsec-ethz/scion-apps
cd ~/go/src/github.com/netsec-ethz/scion-apps/
./deps.sh
make install
```

Get Watcher:
```shell
go get github.com/canthefason/go-watcher
go install github.com/canthefason/go-watcher/cmd/watcher
```

Development Run:
```shell
cd ~/go/src/github.com/netsec-ethz/scion-apps/webapp
watcher \
-a 0.0.0.0 \
-p 8080 \
-r /var/lib/scion/webapp/web/data \
-srvroot $GOPATH/src/github.com/netsec-ethz/scion-apps/webapp/web \
-sabin $GOPATH/bin \
-sroot /etc/scion \
-sbin /usr/bin \
-sgen  /etc/scion/gen \
-sgenc /var/lib/scion \
-slogs /var/log/scion
```

Useful URLs Firefox:
- <about:webrtc>

Useful URLs Chrome:
- <chrome://webrtc-internals>
