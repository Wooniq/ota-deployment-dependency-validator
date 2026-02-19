module github.com/Wooniq/ota-agent/server

go 1.25.6

// 로컬 internal 모듈을 바라보도록 설정
replace github.com/Wooniq/ota-agent/internal => ../../internal

require (
	github.com/Wooniq/ota-agent/internal v0.0.0
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/SAP/go-hdb v1.12.5 // indirect; HANA DB 드라이버 버전 확인 필요
	github.com/eclipse/paho.mqtt.golang v1.5.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/crypto v0.42.0 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)
