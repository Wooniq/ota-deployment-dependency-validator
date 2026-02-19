module github.com/Wooniq/ota-agent/agent

go 1.25.6

// 로컬 internal 모듈을 바라보도록 설정
replace github.com/Wooniq/ota-agent/internal => ../../internal

require (
	github.com/Wooniq/ota-agent/internal v0.0.0
	github.com/eclipse/paho.mqtt.golang v1.5.1
)

require (
	github.com/SAP/go-hdb v1.15.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)
