# phantom-http-server

테스트/개발용 **가상 HTTP/HTTPS 서버 및 RESTful API 시뮬레이터**. 설정 파일에 정의한 API 엔드포인트로 요청을 수신하고, 수신 데이터를 파일·콘솔에 로그로 남기며, 웹 GUI에서 실시간으로 확인할 수 있습니다.

## 주요 기능

* **gin** 기반 fullstack 단일 바이너리 (REST API + HTML GUI)
* **slog** 기반 파일 로그 (stdout 미러링)
* **`setting.yml`** 단일 설정 파일 (HTTP/HTTPS, 포트, API 목록, 로그)
* 설정 파일 경로는 환경변수 **`H2H_SETTING_FILE`** 로 지정
* API별 수신 데이터를 구조화 로그로 파일·콘솔 출력
* `/` GUI에서 API URL별 실시간 로그 스트림(SSE) 및 필터
* **Docker** 기반 실행, VSCode/Cursor **F5** 디버깅 지원
* **dist/** 데몬 실행 스크립트 (`run.sh`, `stop.sh`, `status.sh`)

## 디렉토리 구조

```
cmd/server/           진입점 (main.go)
internal/config/      환경변수 (설정 파일 경로)
internal/settings/    setting.yml 로더
internal/logging/     slog 파일 로거
internal/logbuffer/   인메모리 요청 로그 + SSE 구독
internal/server/      gin 라우터/핸들러
web/                  HTML/CSS/JS GUI (embed)
settings.example/     설정 파일 예시
dist/                 배포용 스크립트 (run/stop/status)
```

## 환경변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `H2H_SETTING_FILE` | `./setting.yml` | YAML 설정 파일 경로 |

## 설정 (setting.yml)

`settings.example/setting.yml`을 참고하세요.

```yaml
server:
  port: 8080
  tls:
    enabled: false          # true 이면 HTTPS
    cert_file: ./certs/server.crt
    key_file: ./certs/server.key

log:
  file: ./logs/phantom-http-server.log
  level: info               # debug/info/warn/error

apis:
  - path: /alert-manager/hook1
    methods: [GET, POST, PUT, DELETE]
    description: Alert Manager webhook hook 1
```

## 실행 (Docker)

```bash
cp settings.example/setting.yml ./setting.yml
docker compose up --build
```

* GUI: http://localhost:8080/
* API 예: http://localhost:8080/alert-manager/hook1

## 로컬 실행

```bash
cp settings.example/setting.yml ./setting.yml
go run ./cmd/server
```

## 배포 (dist 데몬)

```bash
./dist.sh
./dist/run.sh
./dist/status.sh
./dist/stop.sh
```

## 디버깅 (F5)

`.vscode/launch.json`이 포함되어 있습니다. `H2H_SETTING_FILE=./setting.yml` 로 VSCode/Cursor에서 **F5** 디버깅할 수 있습니다.

## API 요약

| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/` | 실시간 로그 GUI |
| GET | `/api/status` | 서비스/설정 현황 |
| GET | `/api/stats` | 사용 통계 |
| GET | `/api/apis` | 설정된 API 목록 |
| GET | `/api/logs` | 요청 로그 조회 (필터: `path`, `method`, `q`) |
| GET | `/api/logs/stream` | SSE 실시간 로그 스트림 |
| * | `/alert-manager/hook1` | Alert Manager webhook (설정에 따라 추가 가능) |

## About

Virtual HTTP/HTTPS server for testing and development. [phantom-exporter](https://github.com/bonukr/phantom-exporter) 프로젝트 구조를 참고했습니다.
