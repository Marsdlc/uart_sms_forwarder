# 短信UART转发器

基于 合宙Air780 XXX 系列设备的短信转发系统，支持接收短信并通过串口转发到上位机。

[项目说明](https://blog.typesafe.cn/posts/air780e-giffgaff/)

**已测试设备**

- Air780EHV
- Air780EHM
- Air780E (可以使用，但属于过时设备，不建议购买)
- Air780EPV (可以使用，但属于过时设备，不建议购买)


## 🌟 功能特性

- 短信转发
- 短信转发前正则过滤，支持匹配后转发或匹配后拦截
- 短信记录
- 发送短信
- 来电通知
- 支持钉钉、企业微信、飞书、自定义 webhook、邮箱通知
- 计划任务发送短信

### 短信过滤

在 Web 控制台的「短信过滤」页面设置规则，默认关闭，对所有已启用的通知渠道统一生效。

- **匹配后转发**：只有正文匹配正则的短信会转发，例如 `验证码|校验码`。
- **匹配后拦截**：正文匹配正则的短信不转发，其余短信正常转发。
- 规则匹配原始短信正文中任意位置，号码和通知附加的时间不参与匹配。被过滤的短信仍保留在短信中心。
- 使用 Go 正则语法，直接填写表达式，无需 `/` 分隔符。可用 `^`、`$` 限定开头和结尾，`(?i)` 忽略大小写，`(?s)` 让点号匹配换行；不支持前后查找和反向引用。
- 「测试当前规则」使用当前草稿和后端同一套匹配逻辑，不需要先保存，也不会发送通知或保存测试短信。保存时会校验正则是否合法。
- 来电、飞行模式、短信发送失败以及通知渠道测试不受短信过滤影响。
- 运行中若配置读取失败或规则损坏，会跳过该条短信的转发并记录日志；短信记录仍会保存。

配置以 `sms_filter_config` 存储在现有属性表中，保存后无需重启。规则测试接口为 `POST /api/sms-filter/test`，
请求格式为 `{"config":{"enabled":true,"mode":"include","pattern":"验证码|校验码"},"content":"您的验证码是123456"}`，
返回 `{"matched":true,"forward":true}`，需要登录认证。

## 截图

![s1.png](screenshots/s1.png)
![s2.png](screenshots/s2.png)
![s3.png](screenshots/s3.png)
![s4.png](screenshots/s4.png)
![s5.png](screenshots/s5.png)

## 🚀 快速开始

----

**重要说明：合宙某些版本的固件有 bug，如果遇到不能收发短信的情况，请换一个固件。**

**重要说明：合宙某些版本的固件有 bug，如果遇到不能收发短信的情况，请换一个固件。**

**重要说明：合宙某些版本的固件有 bug，如果遇到不能收发短信的情况，请换一个固件。**

----

### 1. 硬件准备

**设备准备**：
- 插入有效的SIM卡
- 通过USB连接电脑

### 2. 烧录 Lua 脚本

使用 [**LuaTools**](https://docs.openluat.com/air780epm/common/Luatools/) 烧录 `main.lua` 脚本，第一次烧录需要点击 「下载底层和脚本」

![write.png](screenshots/write.png)

### 3. 测试

![test.png](screenshots/test.png)

### 4. 把设备插入到你的小主机等 Linux USB上


### 5. 运行上位机程序

#### docker 方式安装

```shell
# 创建空目录
mkdir /opt/uart_sms_forwarder
# 下载 docker-compose.yml 文件
wget https://raw.githubusercontent.com/dushixiang/uart_sms_forwarder/main/docker-compose.yml -O /opt/uart_sms_forwarder/docker-compose.yml
# 下载 config.example.yaml 文件
wget https://raw.githubusercontent.com/dushixiang/uart_sms_forwarder/main/config.example.yaml -O /opt/uart_sms_forwarder/config.yaml
```

修改 `docker-compose.yml` 和 `config.yaml` 文件，主要是映射 USB 路径和修改密码。

启动服务

```shell
docker-compose up -d
```

打开浏览器访问 8080 端口。

----

#### 原生方式安装

下载

```shell
wget https://github.com/dushixiang/uart_sms_forwarder/releases/latest/download/uart_sms_forwarder-linux-amd64.tar.gz
```

解压
```bash
tar -zxvf uart_sms_forwarder-linux-amd64.tar.gz -C /opt/
mv /opt/uart_sms_forwarder-linux-amd64 /opt/uart_sms_forwarder
```

创建系统服务

```shell
cat <<EOF > /etc/systemd/system/uart_sms_forwarder.service
[Unit]
Description=uart_sms_forwarder service
After=network.target

[Service]
User=root
WorkingDirectory=/opt/uart_sms_forwarder
ExecStart=/opt/uart_sms_forwarder/uart_sms_forwarder
TimeoutSec=0
RestartSec=10
Restart=always
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF
```

创建 sqllite 目录

```shell
mkdir /opt/uart_sms_forwarder/data
```

启动服务

```shell
systemctl daemon-reload
systemctl enable uart_sms_forwarder
systemctl start uart_sms_forwarder
```

打开浏览器访问 8080 端口。

修改密码等配置项，请参考 [config.example.yaml](config.example.yaml) 文件。


