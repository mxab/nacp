#!/bin/sh
# https://notaryproject.dev/docs/user-guides/installation/uninstall/#remove-the-test-key-and-self-signed-certificate
echo "Deleting test certs on macOS"
NAME="wabbit-networks.io"


notation key delete $NAME
notation cert delete --type ca --store ${NAME} ${NAME}.crt

#echo "rm \"${NOTATION_DIR}/localkeys/${NAME}.key\""


rm "${HOME}/Library/Application Support/notation/localkeys/${NAME}.key"
rm "${HOME}/Library/Application Support/notation/localkeys/${NAME}.crt"
