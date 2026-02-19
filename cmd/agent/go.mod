module github.com/Wooniq/ota-agent/agent

go 1.25.6

// 로컬 internal 모듈을 바라보도록 설정
replace github.com/Wooniq/ota-agent/internal => ../../internal

require (
	github.com/Wooniq/ota-agent/internal v0.0.0
	github.com/eclipse/paho.mqtt.golang v1.5.1
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
)
