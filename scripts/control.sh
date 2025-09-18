#!/bin/bash

# 定义变量
APP_NAME="bin/solana-bot"   # 可执行文件的名称
LOG_FILE="./out.log"          # 日志文件
PID_FILE="app.pid"             # 保存进程 PID 的文件

# 检查程序是否正在运行
is_running() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p "$PID" > /dev/null 2>&1; then
            return 0  # 正在运行
        else
            return 1  # PID 文件存在，但进程未运行
        fi
    else
        return 1      # PID 文件不存在
    fi
}

# 启动程序
start() {
    if is_running; then
        echo "$APP_NAME is already running with PID $(cat $PID_FILE)."
    else
        echo "Starting $APP_NAME..."
        export GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn
        nohup ./"$APP_NAME" bot --mint --smart > "$LOG_FILE" 2>&1 &
        echo $! > "$PID_FILE"
        echo "$APP_NAME started with PID $(cat $PID_FILE)."
    fi
}

# 停止程序
stop() {
    if is_running; then
        echo "Stopping $APP_NAME..."
        kill -9 $(cat "$PID_FILE") && rm -f "$PID_FILE"
        echo "$APP_NAME stopped."
    else
        echo "$APP_NAME is not running."
    fi
}

# 重启程序
restart() {
    echo "Restarting $APP_NAME..."
    stop
    sleep 1
    start
    sleep 1
    status
}

# 查看程序运行状态
status() {
    if is_running; then
        echo "$APP_NAME is running with PID $(cat $PID_FILE)."
    else
        echo "$APP_NAME is not running."
    fi
}

# 帮助信息
help() {
    echo "Usage: $0 {start|stop|restart|status|help}"
    echo "Control the $APP_NAME application."
    echo ""
    echo "Commands:"
    echo "  start    Start the application"
    echo "  stop     Stop the application"
    echo "  restart  Restart the application"
    echo "  status   Check if the application is running"
    echo "  help     Display this help message"
}

# 主逻辑，根据输入参数调用对应函数
case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    help|*)
        help
        ;;
esac