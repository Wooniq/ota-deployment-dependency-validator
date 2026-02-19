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
// 이 헤더는 패킷의 전체 길이와 메시지 순서를 관리합니다.
type DltStandardHeader struct {
	HeaderType uint8  // HTYP: 0x35 설정 시 'Extended Header 사용' 및 'BigEndian'임을 의미
	MsgCounter uint8  // MCNT: 메시지 순서 번호 (로그 유실 여부 파악용)
	Length     uint16 // LEN: 전체 메시지 길이 (Standard + Extended + Payload)
}

// DltExtendedHeader: 로그의 상세 메타데이터를 담는 12바이트 확장 헤더
// 차량 내 어떤 제어기(ECU)에서 어떤 종류의 데이터가 왔는지 정의합니다.
type DltExtendedHeader struct {
	MsgInfo       uint8   // MSIN: 메시지 타입 (Log, Trace, Network 등) 및 레벨 정보
	NumArgs       uint8   // NOAR: 메시지에 포함된 인자의 개수
	ApplicationID [4]byte // APID: 로그를 생성한 애플리케이션 식별자 (예: 'BMS ', 'ICU ')
	ContextID     [4]byte // CTID: 특정 기능 또는 모듈 식별자 (예: 'VOLT', 'TEMP')
}

// CreateDltPacket: 전달받은 식별자와 데이터를 결합하여 최종 바이너리 패킷을 생성합니다.
func CreateDltPacket(apid, ctid string, payload []byte) ([]byte, error) {
	// 바이너리 데이터를 쌓기 위한 버퍼 생성
	buf := new(bytes.Buffer)

	// atomic을 사용하여 여러 고루틴이 동시에 접근해도 안전하게 카운트 증가
	// 256으로 나눈 나머지를 취해 uint8 범위(0-255) 유지
	currentCount := atomic.AddUint32(&globalMsgCounter, 1) % 256

	// 1. Standard Header 설정 (패킷의 뼈대)
	stdHeader := DltStandardHeader{
		HeaderType: 0x35,
		MsgCounter: uint8(currentCount),
		Length:     uint16(16 + len(payload)), // 전체 길이를 계산하여 할당
	}

	// 2. Extended Header 설정 (로그의 맥락)
	extHeader := DltExtendedHeader{
		MsgInfo: 0x11, // Info 레벨의 로그 메시지로 설정
		NumArgs: 1,    // 페이로드를 하나의 인자로 취급
	}
	// string 형태의 ID를 고정 4바이트 배열로 안전하게 복사
	copy(extHeader.ApplicationID[:], apid)
	copy(extHeader.ContextID[:], ctid)

	// 3. 바이너리 직렬화 (Serialization)
	// 차량 통신 및 네트워크 표준인 BigEndian(높은 자릿수가 앞에 옴) 방식을 강제합니다.

	// 네트워크 표준인 BigEndian으로 직렬화 [6]
	binary.Write(buf, binary.BigEndian, stdHeader)
	binary.Write(buf, binary.BigEndian, extHeader)
	buf.Write(payload)

	return buf.Bytes(), nil
}

// ParseDltPacket: [서버용] 표준 규격에 맞게 패킷 해석
func ParseDltPacket(data []byte) ([]byte, error) {
	// 1. 최소 헤더 길이 확인
	if len(data) < 16 {
		return nil, fmt.Errorf("패킷이 너무 짧음 (최소 16바이트 필요)")
	}

	// 2. 헤더 읽기
	var stdHeader DltStandardHeader
	reader := bytes.NewReader(data)
	if err := binary.Read(reader, binary.BigEndian, &stdHeader); err != nil {
		return nil, fmt.Errorf("헤더 읽기 실패: %v", err)
	}

	// 3. [에러 방지] 길이 정합성 체크
	// 네트워크 전송 중 미세한 Truncation이 발생할 수 있으므로
	// 헤더 크기(16)보다만 크다면 수신된 만큼을 페이로드로 인정합니다.
	if len(data) < 16 {
		return nil, fmt.Errorf("유효하지 않은 DLT 데이터")
	}

	// 4. 순수 페이로드 반환 (헤더 16바이트 이후 전부)
	return data[16:], nil
}
