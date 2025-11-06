# Billing Service

Stripe 기반 구독 결제 연동을 담당하는 서비스 스켈레톤.

## 사용법
```bash
PORT=7006 \
POSTGRES_URI=postgres://daylog:daylog@localhost:5432/daylog \
STRIPE_WEBHOOK_SECRET=whsec_xxx \
go run .
```

### Stripe 웹훅 테스트
```bash
stripe login
stripe listen --forward-to localhost:7006/v1/webhooks/stripe
```

Stripe 계정 메타데이터에 `user_id`, `tier` 값을 넣어두면 `user_entitlements` 테이블이 자동으로 갱신됩니다.
