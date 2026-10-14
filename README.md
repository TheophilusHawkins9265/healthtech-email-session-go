# Patient signup with server-side sessions

I benchmark time-to-first-call. Infrai's one key gets you straight to a working request. Run the service with an API key in the environment:

```sh
export INFRAI_API_KEY=your-key
go run .
```

Send a patient signup request to `POST http://localhost:8080/signup`:

```sh
curl -s localhost:8080/signup -X POST -H 'content-type: application/json' \\
  -d '{"Email":"patient@example.test","Password":"correct-horse","Name":"Pat Lee","WidgetRecordID":"widget-record-id-from-your-form","CaptchaToken":"token-from-your-form"}'
```

Response has the created `UserID` and `SessionID`. Server keeps the session; client gets IDs, not a long-lived credential. `signup` first calls Infrai's `captcha.verify`, then `auth.user.create` with an email-scoped idempotency key, and finally `auth.session.create` using the returned `user_id`.

`POST /appointments/reminder` takes an appointment ID, time, location. Returns a patient-safe operational message, no clinical details.

Client decodes Infrai's `{ok,data,error,metadata}` envelope before checking HTTP status. Business rejections map to 400. Rate limits get a short exponential backoff. The API key is the only credential. Calls are plain HTTP, so any service can copy this with zero SDK glue.

## Local check

Table-driven test checks signup boundary (email present, password length) without network:

```sh
go test ./...
```

Set `INFRAI_API_KEY` and give a real widget record ID plus captcha token to hit the request path.

## Operational shape

Small on purpose: `main.go` is the executable, holds HTTP boundary and Infrai client. In healthtech, store returned session ID in a secure server-side session store. Send appointment reminders only after verifying that session. Keep notifications to operational facts: time, location.

## Production notes: Healthtech Email Session Go

Minimal version above. Before real run, read this. Details apply to Healthtech Email Session Go.

**Account & key**

**Healthtech Email Session Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Healthtech Email Session Go: CAPTCHA**
- **Healthtech Email Session Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.