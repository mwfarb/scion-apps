#!/bin/bash

mkdir ${SCION_GEN}/ISD${ISD}/AS${AS}/sig${IA}-1

# Create script to set interfaces and run the SIG
file=${SCION_GEN}/ISD${ISD}/AS${AS}/sig${IA}-1/run_sig${IA}.sh
cat >$file <<EOL
#!/bin/bash

# Webapp SIG Server Test

# Do not edit, this file is autogenerated by webapp.

# WARNING: the SIG binary must be built beforehand to this location:
# ${SCION_BIN}/sig

# Create the configuration directories for the SIGs,
mkdir -p ${SCION_GEN}/ISD${ISD}/AS${AS}/sig${IA}-1/

# set the linux capabilities on the binary:
sudo setcap cap_net_admin+eip ${SCION_BIN}/sig

# Enable routing:
sudo sysctl net.ipv4.conf.default.rp_filter=0
sudo sysctl net.ipv4.conf.all.rp_filter=0
sudo sysctl net.ipv4.ip_forward=1

#create two dummy interfaces:
sudo modprobe dummy

sudo ip link add dummy${IdRemote} type dummy
sudo ip addr add 172.16.0.${IdRemote}/32 brd + dev dummy${IdRemote} label dummy${IdRemote}:0

# Now we need to add the routing rules for the two SIGs:
sudo ip rule add to 172.16.${IdLocal}.0/24 lookup ${IdRemote} prio ${IdRemote}

# Now start the two SIGs with the following commands:
${SCION_BIN}/sig -config=${SCION_GEN}/ISD${ISD}/AS${AS}/sig${IA}-1/sig${IA}.config > ${SCION_LOGS}/sig${IA}-1.log 2>&1 &

# To show the ip rules and routes
sudo ip rule show
sudo ip route show table ${IdLocal}
sudo ip route show table ${IdRemote}

# Add some server on host Local:
sudo ip link add server type dummy
sudo ip addr add 172.16.${IdLocal}.1/24 brd + dev server label server:0

mkdir ${STATIC_ROOT}/data/www/sig${IA}
cd ${STATIC_ROOT}/data/www/sig${IA}
echo "Hello World from ${IAd}!" > ${STATIC_ROOT}/data/www/sig${IA}/sighello.html

# The 'python3 -m http.server --bind' command will be run by webapp...
EOL
cat $file

# Create script to test the running SIG
file=${SCION_GEN}/ISD${ISD}/AS${AS}/sig${IA}-1/test_sig${IA}.sh
cat >$file <<EOL
#!/bin/bash
python3 -m http.server --bind 172.16.${IdLocal}.1 ${ServePort}
EOL
cat $file
