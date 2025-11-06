# 키 관리

로컬 개발 환경에서는 임시 키를 사용하며, 실제 운영 환경에서는 AWS Secrets Manager 등 외부 비밀 저장소를 사용해야 한다.

예시:
```bash
openssl genrsa -out dev_private.pem 2048
openssl rsa -in dev_private.pem -pubout -out dev_public.pem
```

`.env` 파일의 `AUTH_PUBLIC_KEY_PATH`가 이 경로를 가리키도록 설정한다.
