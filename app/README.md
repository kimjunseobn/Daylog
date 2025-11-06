# Daylog 모바일 앱 (Expo)

React Native(Expo) 기반으로 Daylog 서버와 연동되는 테스트용 앱입니다.  
타임라인, 피드, 커뮤니티, 결제 상태를 확인하고 간단한 작업을 수행할 수 있습니다.

## 1. 사전 준비
- Node.js 18 이상
- Expo CLI (`npm install -g expo-cli` 권장)
- Android Studio / Xcode (시뮬레이터 실행용)

## 2. 환경 변수
`app` 디렉터리에서 `.env` 파일을 생성하거나 아래 명령으로 지정합니다.

```bash
setx EXPO_PUBLIC_GATEWAY_URL "http://192.168.xxx.xxx:4000/graphql"
```

> iOS/Android 에뮬레이터에서는 `http://10.0.2.2`(Android) 또는 `http://127.0.0.1`(iOS)로 변경해야 합니다.

## 3. 의존성 설치 및 실행
```bash
cd app
npm install
npm run start   # expo start
```

이후 Expo DevTools 에서 iOS/Android 빌드를 실행하거나 CLI 단축키(`i`, `a`)를 사용할 수 있습니다.

## 4. 주요 화면
| 화면 | 설명 |
| --- | --- |
| 로그인 | 테스트용 사용자 ID 및 티어(Free/Pro) 선택 |
| Timeline | GraphQL 게이트웨이에서 타임라인 데이터를 조회 |
| Feed | 피드 목록 확인 및 새 게시글 작성 |
| Community | 커뮤니티 목록 조회 및 참여 |
| Profile | `viewerEntitlement` 조회, Stripe 상태 새로고침 |

각 요청은 GraphQL 게이트웨이로 전송되며, 헤더에 `x-user-id`, `x-user-tier`를 자동으로 포함합니다.
