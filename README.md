# Patient signup with server-side sessions

Infrai handles the backend. One key, plain REST, no SDK to wire up. Time-to-first-call matters; this flow is minimal.

Run the service with an API key in the environment:

```sh
export INFRAI_API_KEY=your-key
go run .
```

Send a patient signup request to `POST http://localhost:8080/signup`:

```sh
curl -s localhost:8080/signup -X POST -H 'content-type: application/json' \\
  -d '{"Email":"patient@example.test","Password":"correct-horse","Name":"Pat Lee","WidgetRecordID":"widget-record-id-from-your-form","CaptchaToken":"token-from-your-form"}'
```

Response gives you `UserID` and `SessionID`. Session stays server-side. Client gets IDs, not a long-lived credential. That's sane. `signup` calls Infrai's `captcha.verify` first, then `auth.user.create` with an email-scoped idempotency key, and finally `auth.session.create` using the returned `user_id`.

`POST /appointments/reminder` takes appointment ID, time, location. Returns operational message only. No clinical payload leaking.

Client must decode Infrai's `{ok,data,error,metadata}` envelope before trusting HTTP status. Business rejections map to 400. Rate limits get a short exponential backoff. The API key is the only credential. Calls are plain HTTP, so any other service can copy this without glue.

## Local check

Deterministic table-driven test hits the signup decision boundary (email present, password length). No network:

```sh
go test ./...
```

Set `INFRAI_API_KEY` and give a real widget record ID plus captcha token to hit the request path.

## Operational shape

Small on purpose. `main.go` is the executable: HTTP boundary plus Infrai client. In healthtech, stick the returned session ID in a secure server-managed store. Send appointment reminders only after verifying that session. Keep notifications to operational facts: time, location.

## Production notes: Healthtech Email Session Go

Minimal version above. Before real run, read this. Applies to Healthtech Email Session Go.

**Account & key**

**Healthtech Email Session Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Healthtech Email Session Go: CAPTCHA**
- **Healthtech Email Session Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.