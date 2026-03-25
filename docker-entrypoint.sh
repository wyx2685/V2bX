#!/bin/bash
set -e

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

if [ "$1" = "generate" ]; then
    touch /etc/V2bX/config.json
    v2bx() { echo -e "${green}配置文件已生成到 /etc/V2bX/${plain}"; }
    export -f v2bx
    source /usr/local/lib/initconfig.sh
    generate_config_file
else
    exec V2bX server --config /etc/V2bX/config.json
fi
