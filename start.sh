
# 域名定时更新 DDNS
/overlay/wz/dnsup >/dev/null 2>&1 &

# 单文件
#/overlay/wz/xray run -c /overlay/wz/config.json >/dev/null 2>&1 &

# 多文件启动
/overlay/wz/xray run -confdir confs >/dev/null 2>&1 &
