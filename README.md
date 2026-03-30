# MyEvent Backend

Implementacao do backend MVP em Go, cobrindo as fases 1 a 4.

## Entregue

- autenticacao com cadastro, login e `GET /v1/auth/me`
- CRUD privado de eventos
- endpoint publico `GET /v1/public/events/:slug`
- CRUD de convidados por evento
- RSVP publico com listagem privada por evento
- dashboard inicial com contadores de convidados
- payload de QR code por convidado
- check-in e lista de convidados com status de entrada
- presentes, reservas e fluxo de Pix manual
- upload autenticado com suporte a Cloudflare R2 e fallback local
- presets de tema e validacoes extras de URL
- validacoes basicas, JWT e CORS
- persistencia em memoria, pronta para troca por Postgres via interfaces

## Como rodar

```bash
go mod tidy
go run ./cmd/api
```

Servidor padrao em `http://localhost:8080`.

Se as variaveis do R2 nao estiverem preenchidas, os uploads ficam em `LOCAL_UPLOAD_DIR`
e sao servidos em `http://localhost:8080/uploads/...`.
