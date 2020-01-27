#!/bin/bash

cfgdir=${SCION_GEN}/ISD${ISD}/AS${AS}/sig${IA}-1
mkdir ${cfgdir}

file=${cfgdir}/sig${IA}.json
cat >$file <<EOL
{
    "ASes": {
        "${IaRemote}": {
            "Nets": [
                "172.16.${IdRemote}.0/24"
            ],
            "Sigs": {
                "remote-1": {
                    "Addr": "${IpRemote}",
                    "CtrlPort": ${CtrlPortRemote},
                    "EncapPort": ${EncapPortRemote}
                }
            }
        }
    },
    "ConfigVersion": 9001
}
EOL
cat $file
