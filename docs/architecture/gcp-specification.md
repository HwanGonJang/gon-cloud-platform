# Gon Cloud Platform (GCP) 구축 명세서

## 1. 프로젝트 개요

### 1.1 목적

AWS EC2와 유사한 기능을 제공하는 소규모 클라우드 플랫폼 구축

### 1.2 MVP 기능

- VPC 기반 가상 네트워크 환경
- 퍼블릭 서브넷, 라우팅 테이블, 인터넷 게이트웨이
- EC2 유사 가상 인스턴스 생성 및 관리
- 보안 그룹 및 네트워크 ACL
- 웹 기반 관리 콘솔

### 1.3 제약 사항

- 우분투 22.04 기반 3대 서버 구성
- 프라이빗 서브넷은 추후 구현
- 외부에서 인스턴스 직접 접속 가능

## 2. 시스템 아키텍처

### 2.1 서버 구성

```
Control Plane (121.36.55.1)
├── API Server
├── Web Console
├── Database (PostgreSQL)
├── Message Broker (RabbitMQ)
└── Network Controller

Worker Node 1 (121.36.55.2)
├── Hypervisor (KVM/QEMU)
├── Container Runtime (Docker)
├── Network Agent
└── Storage Agent

Worker Node 2 (121.36.55.3)
├── Hypervisor (KVM/QEMU)
├── Container Runtime (Docker)
├── Network Agent
└── Storage Agent
```

### 2.2 네트워크 토폴로지

```
Internet
    │
    ▼
[Internet Gateway]
    │
    ▼
[Virtual Router/NAT]
    │
    ▼
[Open vSwitch Bridge]
    │
    ├── [VPC-1] ── [Subnet-1] ── [Instance-1, Instance-2, ...]
    ├── [VPC-2] ── [Subnet-2] ── [Instance-3, Instance-4, ...]
    └── [VPC-N] ── [Subnet-N] ── [Instance-N, ...]
```

## 3. 기술 스택

### 3.1 백엔드

- **언어**: Go 1.21+
- **API 프레임워크**: Gin
- **데이터베이스**: PostgreSQL 14+
- **메시지 큐**: RabbitMQ 3.12+
- **분산 저장소**: etcd 3.5+

### 3.2 가상화 및 네트워킹

- **컨테이너**: Docker 24.0+
- **하이퍼바이저**: KVM/QEMU
- **네트워킹**: Open vSwitch 3.0+
- **방화벽**: iptables/netfilter

### 3.3 프론트엔드

- **프레임워크**: React 18
- **상태 관리**: Redux Toolkit
- **UI 라이브러리**: Ant Design
- **빌드 도구**: Vite

## 4. 데이터베이스 스키마

### 4.1 핵심 테이블 구조

```sql
-- 사용자 관리
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- VPC 관리
CREATE TABLE vpcs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    name VARCHAR(100) NOT NULL,
    cidr_block CIDR NOT NULL,
    state VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 서브넷 관리
CREATE TABLE subnets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vpc_id UUID REFERENCES vpcs(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    cidr_block CIDR NOT NULL,
    availability_zone VARCHAR(20) NOT NULL,
    subnet_type VARCHAR(20) DEFAULT 'public',
    created_at TIMESTAMP DEFAULT NOW()
);

-- 인스턴스 관리
CREATE TABLE instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    subnet_id UUID REFERENCES subnets(id),
    name VARCHAR(100) NOT NULL,
    instance_type VARCHAR(50) NOT NULL,
    image_id VARCHAR(100) NOT NULL,
    state VARCHAR(20) DEFAULT 'pending',
    private_ip INET,
    public_ip INET,
    worker_node VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 보안 그룹 관리
CREATE TABLE security_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vpc_id UUID REFERENCES vpcs(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 보안 그룹 규칙
CREATE TABLE security_group_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    security_group_id UUID REFERENCES security_groups(id) ON DELETE CASCADE,
    direction VARCHAR(10) NOT NULL, -- 'inbound' or 'outbound'
    protocol VARCHAR(10) NOT NULL,  -- 'tcp', 'udp', 'icmp', 'all'
    port_range VARCHAR(20),         -- '80', '80-443', 'all'
    source_destination CIDR,
    action VARCHAR(10) DEFAULT 'allow',
    created_at TIMESTAMP DEFAULT NOW()
);

-- 라우팅 테이블
CREATE TABLE route_tables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vpc_id UUID REFERENCES vpcs(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    is_main BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 라우팅 규칙
CREATE TABLE routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_table_id UUID REFERENCES route_tables(id) ON DELETE CASCADE,
    destination_cidr CIDR NOT NULL,
    target_type VARCHAR(20) NOT NULL, -- 'gateway', 'instance', 'nat'
    target_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

-- 네트워크 인터페이스
CREATE TABLE network_interfaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID REFERENCES instances(id) ON DELETE CASCADE,
    subnet_id UUID REFERENCES subnets(id),
    private_ip INET NOT NULL,
    mac_address MACADDR NOT NULL,
    is_primary BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## 5. API 엔드포인트 설계

### 5.1 VPC 관리 API

```
POST   /api/v1/vpcs                 # VPC 생성
GET    /api/v1/vpcs                 # VPC 목록 조회
GET    /api/v1/vpcs/:id             # VPC 상세 조회
PUT    /api/v1/vpcs/:id             # VPC 수정
DELETE /api/v1/vpcs/:id             # VPC 삭제

POST   /api/v1/vpcs/:id/subnets     # 서브넷 생성
GET    /api/v1/vpcs/:id/subnets     # 서브넷 목록 조회

```

### 5.2 인스턴스 관리 API

```
POST   /api/v1/instances            # 인스턴스 생성
GET    /api/v1/instances            # 인스턴스 목록 조회
GET    /api/v1/instances/:id        # 인스턴스 상세 조회
PUT    /api/v1/instances/:id        # 인스턴스 수정
DELETE /api/v1/instances/:id        # 인스턴스 삭제

POST   /api/v1/instances/:id/start  # 인스턴스 시작
POST   /api/v1/instances/:id/stop   # 인스턴스 중지
POST   /api/v1/instances/:id/reboot # 인스턴스 재부팅

```

### 5.3 보안 그룹 API

```
POST   /api/v1/security-groups                    # 보안 그룹 생성
GET    /api/v1/security-groups                    # 보안 그룹 목록
GET    /api/v1/security-groups/:id                # 보안 그룹 상세
PUT    /api/v1/security-groups/:id                # 보안 그룹 수정
DELETE /api/v1/security-groups/:id                # 보안 그룹 삭제

POST   /api/v1/security-groups/:id/rules          # 규칙 추가
DELETE /api/v1/security-groups/:id/rules/:rule_id # 규칙 삭제
```

## 6. 서비스 구성 요소

### 6.1 Control Plane 서비스

### API Server (Port: 8080)

```go
// 주요 구조체
type Server struct {
    DB     *sql.DB
    Router *gin.Engine
    Config *Config
    MQ     MessageQueue
}

// 핵심 기능
- RESTful API 제공
- JWT 기반 인증
- 요청 검증 및 라우팅
- 비동기 작업 큐잉
```

### Network Controller (Port: 8081)

```go
// 주요 기능
- VPC 생성 및 관리
- 서브넷 IP 할당
- 라우팅 테이블 관리
- 보안 그룹 규칙 적용
- Open vSwitch 제어
```

### Instance Manager (Port: 8082)

```go
// 주요 기능
- 가상 머신 생성/삭제
- 리소스 할당 관리
- 워커 노드 로드 밸런싱
- 상태 모니터링
```

### 6.2 Worker Node 서비스

### Hypervisor Agent (Port: 9090)

```go
// 주요 기능
- KVM/QEMU 가상 머신 관리
- 리소스 모니터링
- 상태 보고
```

### Network Agent (Port: 9091)

```go
// 주요 기능
- 가상 네트워크 인터페이스 생성
- iptables 규칙 적용
- 트래픽 모니터링
```

### Container Agent (Port: 9092)

```go
// 주요 기능
- Docker 컨테이너 관리
- 이미지 관리
- 볼륨 마운트
```

## 7. 구현 단계별 계획

### Phase 1: 기반 인프라 구축 (2주)

1. **서버 환경 설정**
    - Ubuntu 22.04 기본 설정
    - 필수 패키지 설치 (KVM, Docker, Open vSwitch)
    - 네트워크 브리지 구성
2. **데이터베이스 설계**
    - PostgreSQL 설치 및 구성
    - 스키마 생성 및 초기 데이터
    - 마이그레이션 도구 개발
3. **기본 API 서버**
    - Go 프로젝트 구조 설정
    - Gin 기반 HTTP 서버
    - 데이터베이스 연결 및 ORM 설정

### Phase 2: 네트워크 가상화 구현 (3주)

1. **VPC 기본 기능**
    - VPC 생성/삭제 API
    - CIDR 블록 관리
    - Open vSwitch 브리지 생성
2. **서브넷 관리**
    - 퍼블릭 서브넷 생성
    - IP 주소 할당 관리
    - DHCP 서버 구성
3. **라우팅 및 게이트웨이**
    - 라우팅 테이블 구현
    - 인터넷 게이트웨이 설정
    - NAT 기능 구현

### Phase 3: 인스턴스 가상화 구현 (3주)

1. **KVM/QEMU 통합**
    - 가상 머신 템플릿 생성
    - 인스턴스 생성/시작/중지 API
    - 리소스 할당 관리
2. **Docker 컨테이너 지원**
    - 컨테이너 기반 경량 인스턴스
    - 이미지 관리 시스템
    - 네트워크 연결
3. **스토리지 관리**
    - LVM 기반 볼륨 관리
    - 스냅샷 기능
    - 볼륨 연결/해제

### Phase 4: 보안 및 네트워크 정책 (2주)

1. **보안 그룹**
    - 인바운드/아웃바운드 규칙
    - iptables 규칙 자동 생성
    - 동적 규칙 업데이트
2. **네트워크 ACL**
    - 서브넷 수준 접근 제어
    - 상태 비저장 필터링
    - 규칙 우선순위 관리

### Phase 5: 웹 콘솔 개발 (3주)

1. **React 프론트엔드**
    - 대시보드 UI
    - VPC 관리 인터페이스
    - 인스턴스 관리 콘솔
2. **실시간 모니터링**
    - WebSocket 기반 상태 업데이트
    - 리소스 사용량 차트
    - 로그 뷰어

### Phase 6: 테스트 및 최적화 (2주)

1. **통합 테스트**
    - API 엔드포인트 테스트
    - 네트워크 연결성 검증
    - 부하 테스트
2. **성능 최적화**
    - 데이터베이스 쿼리 최적화
    - 네트워크 성능 튜닝
    - 메모리 사용량 최적화

## 8. 배포 및 운영

### 8.1 서비스 배포

```bash
# Control Plane 배포
sudo systemctl enable gcp-api-server
sudo systemctl enable gcp-network-controller
sudo systemctl enable gcp-instance-manager

# Worker Node 배포
sudo systemctl enable gcp-hypervisor-agent
sudo systemctl enable gcp-network-agent
sudo systemctl enable gcp-container-agent
```

### 8.2 모니터링 및 로깅

- **메트릭 수집**: Prometheus
- **로그 집계**: ELK Stack (Elasticsearch, Logstash, Kibana)
- **알람**: Grafana + AlertManager

### 8.3 백업 및 복구

- **데이터베이스**: pg_dump 기반 백업
- **설정 파일**: etcd 스냅샷
- **인스턴스 이미지**: 스냅샷 자동화

## 9. 보안 고려사항

### 9.1 네트워크 보안

- VPC 간 완전 격리
- 기본 거부 정책
- DDoS 방어 메커니즘

### 9.2 인증 및 권한

- JWT 토큰 기반 인증
- RBAC (Role-Based Access Control)
- API 키 관리

### 9.3 데이터 보안

- 데이터베이스 암호화
- 네트워크 트래픽 암호화 (TLS)
- 인스턴스 볼륨 암호화

## 10. 확장성 고려사항

### 10.1 수평 확장

- Worker 노드 동적 추가
- 로드 밸런서 도입
- 데이터베이스 샤딩

### 10.2 고가용성

- Control Plane 이중화
- 자동 장애 복구
- 데이터 복제

## 11. 참고 자료

### 11.1 기술 문서

- [KVM Virtualization](https://www.linux-kvm.org/)
- [Open vSwitch Documentation](https://docs.openvswitch.org/)
- [Docker Networking](https://docs.docker.com/network/)

### 11.2 AWS 참조 아키텍처

- [Amazon VPC User Guide](https://docs.aws.amazon.com/vpc/)
- [Amazon EC2 User Guide](https://docs.aws.amazon.com/ec2/)

이 명세서는 Gon Cloud Platform의 전체적인 구현 방향을 제시하며, 실제 개발 과정에서 세부 사항은 조정될 수 있습니다.