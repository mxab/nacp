#!/bin/sh
# https://notaryproject.dev/docs/user-guides/installation/uninstall/#remove-the-test-key-and-self-signed-certificate
echo "Deleting test certs on macOS"
# default name or take first argument
NAME="nacp-demo"
if [ -n "$1" ]; then
  NAME=$1
fi


notation key delete $NAME
notation cert delete -y --type ca --store ${NAME} ${NAME}.crt

#echo "rm \"${NOTATION_DIR}/localkeys/${NAME}.key\""


rm "${HOME}/Library/Application Support/notation/localkeys/${NAME}.key"
rm "${HOME}/Library/Application Support/notation/localkeys/${NAME}.crt"
