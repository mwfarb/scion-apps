#!/bin/bash
# Build and run test servers to emulate test endpoints on localhost for webapp

# test bwtest server
echo "Building bwtest server..."
cd ${GOPATH}/src/github.com/netsec-ethz/scion-apps/bwtester/bwtestserver
go install bwtestserver.go
echo "Running test bwtest server..."
bwtestserver -s 1-ff00:0:112,[127.0.0.2]:30100 -sciondFromIA &

# test camera server
echo "Building image server..."
cd ${GOPATH}/src/github.com/netsec-ethz/scion-apps/camerapp/imageserver
go install imageserver.go
echo "Running test image server..."
go run ${GOPATH}/src/github.com/netsec-ethz/scion-apps/webapp/tests/imgtest/imgserver/local-image.go &
imageserver -s 1-ff00:0:112,[127.0.0.2]:42002 -sciondFromIA &

# test sensor server
echo "Building sensor server..."
cd ${GOPATH}/src/github.com/netsec-ethz/scion-apps/sensorapp/sensorserver
go install sensorserver.go
echo "Running test sensor server..."
python3 ${GOPATH}/src/github.com/netsec-ethz/scion-apps/sensorapp/sensorserver/timereader.py | sensorserver -s 1-ff00:0:112,[127.0.0.2]:42003 -sciondFromIA &

# test scmp echo
# dispatcher is responsible for responding to echo

# test scmp traceroute
# dispatcher is responsible for responding to traceroute

# test pingpong server
cd $SC
echo "Running test pingpongserver..."
./bin/pingpong -mode server -local 1-ff00:0:112,[127.0.0.2]:40002 -sciondFromIA &
