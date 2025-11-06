# Billing Service

Stripe 기반 구독 결제 연동을 담당하는 서비스 스켈레톤.

## 사용법
```bash
PORT=7000 go run .
```

### Stripe 웹훅 시뮬레이션
```bash
curl -X POST http://localhost:7000/v1/webhooks/stripe \
  -H "Content-Type: application/json" \
  -d '{"type":"invoice.paid","data":{"object":{"subscription":"sub_xxx"}}}'
```
