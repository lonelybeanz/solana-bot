#!/bin/bash

APP_NAME="bin/solana-bot"
HOST="root@portal"
ROOT_DIR="/root/bot"


# rsync -avz --backup --suffix=".bak_$(date +%Y%m%d%H%M%S)" --delete "$APP_NAME" "$HOST:$ROOT_DIR/bin/"
# rsync -avz --delete $APP_NAME $HOST:$ROOT_DIR/bin/
# rsync -avz --delete etc/etc.yaml $HOST:$ROOT_DIR/etc/
rsync -avz --delete config $HOST:$ROOT_DIR/
# rsync -avz --delete desc $HOST:$ROOT_DIR/
# rsync -avz --delete etc/private.rsa $HOST:$ROOT_DIR/etc/private.rsa
# rsync -avz --delete etc/.env $HOST:$ROOT_DIR/etc/.env
# rsync -avz --delete scripts/control.sh $HOST:$ROOT_DIR/


# echo "restart server"
# ssh $HOST "cd $ROOT_DIR && chmod +x control.sh && service bot restart"
# echo "restart done"