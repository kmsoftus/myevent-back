# MyEvent — Backend técnico (Go)  
Arquitetura inicial, endpoints, entidades, fluxos e variáveis de ambiente.

---

## 1. Objetivo do backend

O backend do **MyEvent** será responsável por:

- autenticação de usuários
- gestão de eventos
- gestão de convidados
- RSVP
- geração e validação de QR code
- check-in
- gestão de presentes
- fluxo de Pix manual
- upload de arquivos para R2
- servir dados para o painel e para o site público

---

## 2. Stack do backend

- **Linguagem:** Go
- **Router HTTP:** chi ou gin
- **Banco:** PostgreSQL
- **Storage:** Cloudflare R2
- **Auth:** JWT
- **Migrations:** golang-migrate ou goose
- **ORM / query builder:** sqlc, ent ou gorm  
  **Recomendação:** `sqlc` ou `gorm`
- **Deploy da API:** Railway
- **Banco Postgres:** Railway
- **Arquivos / imagens:** Cloudflare R2

---

## 3. Estrutura sugerida de pastas

```txt
/backend
  /cmd
    /api
      main.go

  /internal
    /config
    /database
    /http
      /handlers
      /middleware
      /routes
    /services
    /repositories
    /models
    /dto
    /utils
    /auth
    /storage
    /qrcode

  /migrations
  /docs
  go.mod
  go.sum
```

---

## 4. Arquitetura em camadas

### Handlers
Recebem request e retornam response.

### Services
Contêm regras de negócio.

### Repositories
Fazem leitura e escrita no banco.

### Storage
Responsável por upload e remoção de arquivos no R2.

### Auth
Responsável por JWT, hash de senha e middleware de autenticação.

---

## 5. Domínios principais

O backend pode ser separado em módulos:

- auth
- users
- events
- public_events
- guests
- rsvps
- checkin
- gifts
- gift_transactions
- uploads
- themes

---

## 6. Modelagem inicial

## users
```sql
id UUID PK
name TEXT
email TEXT UNIQUE
password_hash TEXT
created_at TIMESTAMP
updated_at TIMESTAMP
```

## events
```sql
id UUID PK
user_id UUID FK -> users.id
title TEXT
slug TEXT UNIQUE
type TEXT
description TEXT
date DATE
time TEXT
location_name TEXT
address TEXT
cover_image_url TEXT
host_message TEXT

theme TEXT
primary_color TEXT
secondary_color TEXT
background_color TEXT
text_color TEXT

pix_key TEXT
pix_holder_name TEXT

status TEXT
created_at TIMESTAMP
updated_at TIMESTAMP
```

## guests
```sql
id UUID PK
event_id UUID FK -> events.id
name TEXT
email TEXT NULL
phone TEXT NULL
invite_code TEXT UNIQUE
qr_code_token TEXT UNIQUE
max_companions INT DEFAULT 0
rsvp_status TEXT DEFAULT 'pending'
checked_in_at TIMESTAMP NULL
created_at TIMESTAMP
updated_at TIMESTAMP
```

## rsvps
```sql
id UUID PK
event_id UUID FK -> events.id
guest_id UUID FK -> guests.id
status TEXT
companions_count INT DEFAULT 0
message TEXT NULL
responded_at TIMESTAMP
created_at TIMESTAMP
updated_at TIMESTAMP
```

## gifts
```sql
id UUID PK
event_id UUID FK -> events.id
title TEXT
description TEXT NULL
image_url TEXT NULL
value_cents INT NULL
external_link TEXT NULL
status TEXT DEFAULT 'available'
allow_reservation BOOLEAN DEFAULT true
allow_pix BOOLEAN DEFAULT true
created_at TIMESTAMP
updated_at TIMESTAMP
```

## gift_transactions
```sql
id UUID PK
gift_id UUID FK -> gifts.id
event_id UUID FK -> events.id
guest_name TEXT
guest_contact TEXT NULL
type TEXT
status TEXT
message TEXT NULL
created_at TIMESTAMP
confirmed_at TIMESTAMP NULL
updated_at TIMESTAMP
```

---

## 7. Status sugeridos

### Event.status
- `draft`
- `published`
- `closed`

### Guest.rsvp_status
- `pending`
- `confirmed`
- `declined`

### Gift.status
- `available`
- `reserved`
- `pending_payment`
- `confirmed`
- `unavailable`

### GiftTransaction.type
- `reservation`
- `pix`

### GiftTransaction.status
- `pending`
- `confirmed`
- `canceled`

---

## 8. Fluxo de autenticação

### Cadastro
- usuário envia nome, email e senha
- backend valida dados
- gera hash da senha
- salva no banco
- opcional: já retorna JWT

### Login
- usuário envia email e senha
- backend valida
- compara senha com hash
- retorna access token JWT

### Middleware auth
- lê `Authorization: Bearer <token>`
- valida assinatura
- injeta `user_id` no contexto da request

---

## 9. Endpoints REST

## Auth

### POST `/v1/auth/register`
Cria conta.

**body**
```json
{
  "name": "Kaleb",
  "email": "kaleb@email.com",
  "password": "12345678"
}
```

**response**
```json
{
  "user": {
    "id": "uuid",
    "name": "Kaleb",
    "email": "kaleb@email.com"
  },
  "token": "jwt"
}
```

---

### POST `/v1/auth/login`
Autentica usuário.

**body**
```json
{
  "email": "kaleb@email.com",
  "password": "12345678"
}
```

---

### GET `/v1/auth/me`
Retorna usuário autenticado.

---

## Events (privado)

### POST `/v1/events`
Cria evento.

**body**
```json
{
  "title": "Casamento Ana & João",
  "slug": "ana-joao",
  "type": "casamento",
  "description": "Nosso grande dia",
  "date": "2026-10-12",
  "time": "16:00",
  "location_name": "Espaço Bela Vista",
  "address": "Rua Exemplo, 123",
  "host_message": "Esperamos vocês",
  "theme": "classic",
  "primary_color": "#2563eb",
  "secondary_color": "#ffffff",
  "background_color": "#f8fafc",
  "text_color": "#111827",
  "pix_key": "email@pix.com",
  "pix_holder_name": "Ana Silva"
}
```

---

### GET `/v1/events`
Lista eventos do usuário autenticado.

---

### GET `/v1/events/:eventId`
Detalha evento do painel.

---

### PATCH `/v1/events/:eventId`
Atualiza dados do evento.

---

### PATCH `/v1/events/:eventId/status`
Atualiza status do evento.

**body**
```json
{
  "status": "published"
}
```

---

### DELETE `/v1/events/:eventId`
Remove evento.

---

## Public Event

### GET `/v1/public/events/:slug`
Retorna dados públicos do evento para o site.

**response sugerida**
```json
{
  "id": "uuid",
  "title": "Casamento Ana & João",
  "slug": "ana-joao",
  "type": "casamento",
  "description": "Nosso grande dia",
  "date": "2026-10-12",
  "time": "16:00",
  "location_name": "Espaço Bela Vista",
  "address": "Rua Exemplo, 123",
  "cover_image_url": "https://...",
  "host_message": "Esperamos vocês",
  "theme": "classic",
  "primary_color": "#2563eb",
  "secondary_color": "#ffffff",
  "background_color": "#f8fafc",
  "text_color": "#111827",
  "status": "published"
}
```

---

## Guests

### POST `/v1/events/:eventId/guests`
Cria convidado.

**body**
```json
{
  "name": "Maria",
  "email": "maria@email.com",
  "phone": "79999999999",
  "max_companions": 2
}
```

---

### GET `/v1/events/:eventId/guests`
Lista convidados do evento.

---

### GET `/v1/events/:eventId/guests/:guestId`
Detalha convidado.

---

### PATCH `/v1/events/:eventId/guests/:guestId`
Atualiza convidado.

---

### DELETE `/v1/events/:eventId/guests/:guestId`
Remove convidado.

---

### POST `/v1/events/:eventId/guests/import`
Importação futura por CSV.  
**Pode ficar fora do MVP inicial**, mas já vale deixar previsto.

---

## RSVP

### POST `/v1/public/events/:slug/rsvp`
Cria ou atualiza RSVP.

**body**
```json
{
  "guest_identifier": "invite-code-ou-email-ou-telefone",
  "status": "confirmed",
  "companions_count": 1,
  "message": "Estaremos lá!"
}
```

**observação**
- o `guest_identifier` pode ser um `invite_code`
- também pode evoluir para email ou telefone
- para o MVP, o melhor é usar `invite_code`

---

### GET `/v1/events/:eventId/rsvps`
Lista RSVPs do evento.

---

## QR Code

### GET `/v1/events/:eventId/guests/:guestId/qrcode`
Retorna payload do QR code do convidado.

**response**
```json
{
  "guest_id": "uuid",
  "qr_code_token": "token-unico",
  "checkin_url": "/checkin/token-unico"
}
```

**observação**
- a imagem do QR pode ser gerada no front
- ou backend pode devolver PNG/SVG depois

---

## Check-in

### POST `/v1/events/:eventId/checkin`
Marca check-in via token ou busca manual.

**body por token**
```json
{
  "qr_code_token": "token-unico"
}
```

**body por busca manual**
```json
{
  "guest_id": "uuid"
}
```

**response**
```json
{
  "success": true,
  "guest": {
    "id": "uuid",
    "name": "Maria",
    "rsvp_status": "confirmed",
    "checked_in_at": "2026-03-30T18:00:00Z"
  }
}
```

---

### GET `/v1/events/:eventId/checkin/guests`
Lista convidados com status de check-in.

---

## Gifts

### POST `/v1/events/:eventId/gifts`
Cria presente.

**body**
```json
{
  "title": "Jogo de panelas",
  "description": "Conjunto inox",
  "value_cents": 25990,
  "allow_reservation": true,
  "allow_pix": true,
  "external_link": ""
}
```

---

### GET `/v1/events/:eventId/gifts`
Lista presentes do painel.

---

### GET `/v1/public/events/:slug/gifts`
Lista presentes públicos.

---

### GET `/v1/events/:eventId/gifts/:giftId`
Detalha presente.

---

### PATCH `/v1/events/:eventId/gifts/:giftId`
Atualiza presente.

---

### DELETE `/v1/events/:eventId/gifts/:giftId`
Remove presente.

---

## Gift Transactions / Pix manual

### POST `/v1/public/events/:slug/gifts/:giftId/reserve`
Reserva presente.

**body**
```json
{
  "guest_name": "Carlos",
  "guest_contact": "79999999999",
  "message": "Vou dar esse presente"
}
```

**efeito esperado**
- cria `gift_transaction`
- marca presente como `reserved`

---

### POST `/v1/public/events/:slug/gifts/:giftId/pix`
Informa pagamento Pix manual.

**body**
```json
{
  "guest_name": "Carlos",
  "guest_contact": "79999999999",
  "message": "Já fiz o Pix"
}
```

**efeito esperado**
- cria `gift_transaction`
- marca presente como `pending_payment`

---

### GET `/v1/events/:eventId/gift-transactions`
Lista transações dos presentes.

---

### PATCH `/v1/events/:eventId/gift-transactions/:transactionId/confirm`
Confirma manualmente.

**body**
```json
{
  "status": "confirmed"
}
```

**efeito esperado**
- marca transação como `confirmed`
- marca presente como `confirmed`

---

### PATCH `/v1/events/:eventId/gift-transactions/:transactionId/cancel`
Cancela transação.

**body**
```json
{
  "status": "canceled"
}
```

**efeito esperado**
- cancela transação
- opcionalmente volta presente para `available`

---

## Uploads

### POST `/v1/uploads`
Upload genérico para R2.

**form-data**
- file
- folder (ex: `events/covers`, `events/gifts`)

**response**
```json
{
  "url": "https://cdn.myevent.com.br/events/covers/file.jpg",
  "key": "events/covers/file.jpg"
}
```

---

### DELETE `/v1/uploads`
Remove arquivo do R2.

**body**
```json
{
  "key": "events/covers/file.jpg"
}
```

---

## Dashboard

### GET `/v1/events/:eventId/dashboard`
Retorna resumo do evento.

**response**
```json
{
  "guests_total": 120,
  "guests_confirmed": 80,
  "guests_pending": 30,
  "guests_declined": 10,
  "checked_in_total": 25,
  "gifts_total": 20,
  "gifts_confirmed": 5,
  "gifts_pending_payment": 3
}
```

---

## 10. Regras importantes de backend

- usuário só pode acessar eventos dele
- slug do evento deve ser único
- `invite_code` deve ser único
- `qr_code_token` deve ser único
- check-in não pode ser duplicado sem alerta
- RSVP só pode acontecer se evento estiver `published`
- presente `confirmed` não pode voltar para disponível sem ação manual
- presente reservado deve bloquear nova reserva, salvo regra futura
- Pix é manual, sem conciliação automática no MVP

---

## 11. Upload para R2

### Estratégia sugerida
A API recebe o arquivo e faz upload para o R2.

### Pastas sugeridas no bucket
```txt
events/covers/
events/gifts/
events/gallery/
```

### URLs
O ideal é servir por:
- domínio público do bucket
- ou um domínio CDN próprio no futuro

---

## 12. Variáveis de ambiente

## App
```env
APP_ENV=development
APP_PORT=8080
APP_BASE_URL=http://localhost:8080
FRONTEND_URL=http://localhost:3000
JWT_SECRET=super-secret
JWT_EXPIRES_IN=168h
```

## PostgreSQL (Railway)
```env
DATABASE_URL=postgres://postgres:password@host:5432/railway
DB_HOST=host
DB_PORT=5432
DB_NAME=railway
DB_USER=postgres
DB_PASSWORD=password
DB_SSLMODE=require
```

**observação**
- se usar `DATABASE_URL`, muitas vezes já basta
- Railway normalmente entrega a connection string pronta
- `sslmode=require` costuma ser necessário em produção

## Cloudflare R2
```env
R2_ACCOUNT_ID=your_account_id
R2_ACCESS_KEY_ID=your_access_key_id
R2_SECRET_ACCESS_KEY=your_secret_access_key
R2_BUCKET=myevent-assets
R2_REGION=auto
R2_ENDPOINT=https://<account_id>.r2.cloudflarestorage.com
R2_PUBLIC_URL=https://pub-xxxxxxxx.r2.dev
```

## CORS
```env
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://myevent.com.br,https://www.myevent.com.br
```

## Opcional
```env
LOG_LEVEL=info
```

---

## 13. Exemplo de `.env.example`

```env
APP_ENV=development
APP_PORT=8080
APP_BASE_URL=http://localhost:8080
FRONTEND_URL=http://localhost:3000

JWT_SECRET=
JWT_EXPIRES_IN=168h

DATABASE_URL=
DB_HOST=
DB_PORT=5432
DB_NAME=
DB_USER=
DB_PASSWORD=
DB_SSLMODE=require

R2_ACCOUNT_ID=
R2_ACCESS_KEY_ID=
R2_SECRET_ACCESS_KEY=
R2_BUCKET=
R2_REGION=auto
R2_ENDPOINT=
R2_PUBLIC_URL=

CORS_ALLOWED_ORIGINS=http://localhost:3000
LOG_LEVEL=info
```

---

## 14. Railway — deploy da API

### Estrutura básica
- serviço 1: API Go
- serviço 2: PostgreSQL Railway

### Build
Se usar Docker:
- Railway sobe com Dockerfile

Se não usar Docker:
- Railway também pode buildar via Nixpacks

### Recomendação
Para Go, eu usaria **Dockerfile** para manter previsível.

### Exemplo de comando da app
```bash
./api
```

---

## 15. Dockerfile sugerido

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/api .
EXPOSE 8080
CMD ["./api"]
```

---

## 16. Fluxo de criação de evento

1. usuário cria conta
2. faz login
3. cria evento
4. sobe capa
5. configura tema e cores
6. adiciona convidados
7. adiciona presentes
8. publica evento
9. compartilha link `/e/slug`

---

## 17. Fluxo de RSVP

1. convidado acessa página do evento
2. entra na confirmação
3. informa identificador
4. confirma presença ou recusa
5. backend grava RSVP
6. painel atualiza contadores

---

## 18. Fluxo de presente com Pix manual

1. convidado acessa lista de presentes
2. escolhe presente
3. reserva ou informa Pix
4. backend cria `gift_transaction`
5. presente vai para `reserved` ou `pending_payment`
6. organizador confirma manualmente no painel
7. presente passa para `confirmed`

---

## 19. Segurança mínima recomendada

- hash de senha com bcrypt
- JWT assinado com secret forte
- rate limit em auth e RSVP
- validação de input
- CORS restrito
- logs de erro
- não expor stack trace em produção

---

## 20. Ordem recomendada de implementação

### Fase 1
- auth
- events
- public event
- guests

### Fase 2
- RSVP
- dashboard
- QR payload

### Fase 3
- check-in
- gifts
- gift transactions

### Fase 4
- upload R2
- ajustes de temas
- melhoria de regras

---

## 21. Decisões recomendadas para o MVP

- usar REST
- usar JWT simples
- usar upload via backend para R2
- usar QR token em vez de QR com payload gigante
- usar Pix manual sem automação
- deixar importação CSV para depois
- deixar refresh token para depois, se quiser simplificar

---

## 22. Próximo arquivo ideal

Depois desse, o melhor próximo documento seria:

- **rotas do front Next**
- ou **schema SQL inicial**
- ou **documentação OpenAPI / Swagger**
