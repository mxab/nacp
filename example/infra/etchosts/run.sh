#!/bin/bash -e


function restore_hosts {

    echo "Restoring hosts file"
    sudo cp /etc/hosts.bak /etc/hosts
    echo "Done"

}

trap restore_hosts EXIT
echo "Backing up /etc/hosts to /etc/hosts.bak"
sudo cp /etc/hosts /etc/hosts.bak
sudo consul-template -template=etchost.tmpl:/etc/hosts
