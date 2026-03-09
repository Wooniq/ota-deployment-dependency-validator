package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync/atomic" // 스레드 안전한 카운팅을 위해 추가
)

// 전역 메시지 카운터 (0~255 순환)
var globalMsgCounter uint32

// DltStandardHeader: 모든 DLT 패킷에 포함되는 4바이트 필수 헤더
type DltStandardHeader struct {
	HeaderType uint8  // HTYP: 0x35 설정 시 'Extended Header 사용' 및 'BigEndian'임을 의미
	MsgCounter uint8  // MCNT: 메시지 순서 번호 (로그 유실 여부 파악용)
	Length     uint16 // LEN: 전체 메시지 길이 (Standard + Extended + Payload)
}

// DltExtendedHeader: 로그의 상세 메타데이터를 담는 10바이트 확장 헤더
type DltExtendedHeader struct {
	MsgInfo       uint8   // MSIN: 메시지 타입 (Log, Trace, Network 등) 및 레벨 정보
	NumArgs       uint8   // NOAR: 메시지에 포함된 인자의 개수
	ApplicationID [4]byte // APID: 로그를 생성한 애플리케이션 식별자 (예: 'BMS ', 'ICU ')
	ContextID     [4]byte // CTID: 특정 기능 또는 모듈 식별자 (예: 'VOLT', 'TEMP')
}

// CreateDltPacket: 전달받은 식별자와 데이터를 결합하여 최종 바이너리 패킷을 생성합니다.
func CreateDltPacket(apid, ctid string, payload []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	currentCount := atomic.AddUint32(&globalMsgCounter, 1) % 256

	stdHeader := DltStandardHeader{
		HeaderType: 0x35,
		MsgCounter: uint8(currentCount),
		Length:     uint16(14 + len(payload)), // 실제 헤더 크기 14바이트 할당
	}

	extHeader := DltExtendedHeader{
		MsgInfo: 0x11,
		NumArgs: 1,
	}
	copy(extHeader.ApplicationID[:], apid)
	copy(extHeader.ContextID[:], ctid)

	binary.Write(buf, binary.BigEndian, stdHeader)
	binary.Write(buf, binary.BigEndian, extHeader)
	buf.Write(payload)

	return buf.Bytes(), nil
}

// ParseDltPacket: [서버용] 표준 규격에 맞게 패킷 해석
func ParseDltPacket(data []byte) ([]byte, error) {
	if len(data) < 14 {
		return nil, fmt.Errorf("패킷이 너무 짧음 (최소 14바이트 필요)")
	}

	var stdHeader DltStandardHeader
	reader := bytes.NewReader(data)
	if err := binary.Read(reader, binary.BigEndian, &stdHeader); err != nil {
		return nil, fmt.Errorf("헤더 읽기 실패: %v", err)
	}

	return data[14:], nil
}
