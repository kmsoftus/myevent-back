# Relatório de Auditoria — Backend MyEvent

> **Data:** 2026-04-01  
> **Escopo:** `myevent-back/` — Go, Chi, PostgreSQL/PGX, JWT, bcrypt, AWS S3/R2, Brevo, Telegram  
> **Metodologia:** Revisão estática de 100% dos arquivos de source, leitura de migrations e testes.

---

## Legenda de Prioridade

| Prioridade | Descrição |
|---|---|
| 🔴 **P0 — Crítico** | Segurança ou integridade de dados em risco imediato |
| 🟠 **P1 — Alto** | Causa bugs funcionais graves ou vetor de abuso |
| 🟡 **P2 — Médio** | Qualidade, resiliência ou UX degradados |
| 🟢 **P3 — Baixo** | Boas práticas, observabilidade, refinamentos |

---

## P0 — Críticos

### 1. Race condition na reserva de presente (sem transação atômica)

**Arquivo:** `internal/services/gift_transaction_service.go` — `createPublicTransaction`

```go
if gift.Status != "available" {
    return nil, fmt.Errorf("%w: Este presente nao esta disponivel.", ErrConflict)
}
// — duas goroutines passam aqui simultaneamente —
if err := s.transactions.Create(ctx, transaction); err != nil { ... }
if err := s.gifts.Update(ctx, gift); err != nil { ... }
```

O fluxo lê `gift.Status`, valida disponibilidade e depois faz dois writes separados — **sem transação de banco de dados**. Duas requisições concorrentes podem ultrapassar a validação e criar duas transações para o mesmo presente (double-booking).

**Impacto:** Dois convidados diferentes recebem confirmação do mesmo presente.

**Correção:** Envolver a verificação de estado e os dois writes em uma única transação PostgreSQL (`BEGIN … COMMIT`) ou usar `SELECT … FOR UPDATE` no gift antes de criar a transação.

---

### 2. Race condition em RSVP aberto (`findOrCreateOpenRSVPGuest`)

**Arquivo:** `internal/services/rsvp_service.go` — `findOrCreateOpenRSVPGuest`

```go
guests, _ := s.guests.ListByEventID(ctx, eventID) // lê todos
for _, g := range guests {
    if strings.ToLower(g.Name) == nameLower { return g, nil }
}
// — janela de corrida aqui —
s.guests.Create(ctx, guest) // cria se não achou
```

Duas submissões simultâneas com o mesmo nome criam dois registros de convidado distintos para o mesmo evento, duplicando o participante.

**Impacto:** Dados de convidado duplicados; RSVP incoerente.

**Correção:** Usar `INSERT … ON CONFLICT DO NOTHING RETURNING id` ou `SELECT … WHERE LOWER(name) = $1 FOR UPDATE` dentro de uma transação.

---

### 3. Confirm/Cancel de transação sem atomicidade

**Arquivo:** `internal/services/gift_transaction_service.go` — `Confirm` / `Cancel`

```go
s.transactions.Update(ctx, transaction) // atualiza status da transação
s.gifts.Update(ctx, gift)               // atualiza status do presente
```

Se o segundo `Update` falhar (ex.: timeout, crash), a transação fica em `confirmed` mas o presente permanece em `reserved` (ou vice-versa). Estado permanentemente inconsistente.

**Correção:** Mesmos dois writes dentro de uma transação PostgreSQL.

---

### 4. JWT_SECRET com valor padrão inseguro

**Arquivo:** `internal/config/config.go:49`

```go
JWTSecret: getEnv("JWT_SECRET", "super-secret"),
```

Se a variável de ambiente `JWT_SECRET` não for definida em produção, todos os tokens serão assinados com `"super-secret"` — chave pública e trivialmente forjável.

**Impacto:** Qualquer pessoa pode gerar tokens válidos e se passar por qualquer usuário.

**Correção:** Remover o fallback e usar `log.Fatal` se `JWT_SECRET` estiver ausente, da mesma forma que já é feito para `DATABASE_URL`.

---

### 5. Ausência de rate limiting nas rotas públicas

**Arquivo:** `internal/http/routes/router.go`

Rotas sem qualquer throttle:
- `POST /v1/auth/login` → brute force de senhas
- `POST /v1/auth/forgot-password` → spam de e-mails de redefinição (custo Brevo)
- `POST /v1/auth/register` → criação em massa de contas falsas
- `POST /v1/public/events/{slug}/rsvp` → RSVP flood
- `POST /v1/public/events/{slug}/gifts/{giftId}/reserve` → spam de reservas

**Correção:** Adicionar middleware de rate limit por IP (ex.: `go-chi/httprate`) nas rotas públicas, com limites distintos por endpoint.

---

## P1 — Altos

### 6. `errors.Is` vs `==` no módulo de reset de senha

**Arquivo:** `internal/services/auth_password_reset.go` — linhas 35 e 97

```go
if err == repositories.ErrNotFound {  // ❌ não unwrapa erros embrulhados
```

Os outros services usam corretamente `errors.Is(err, repositories.ErrNotFound)`. Aqui a comparação direta pode falhar silenciosamente se o repositório embrulhar o error (ex.: num futuro refactor), causando vazamento do erro interno em vez de retornar a mensagem amigável.

**Correção:** Substituir por `errors.Is(err, repositories.ErrNotFound)`.

---

### 7. Ausência de graceful shutdown

**Arquivo:** `cmd/api/main.go`

```go
http.ListenAndServe(":"+cfg.AppPort, router)
```

O servidor não captura `SIGTERM`/`SIGINT` para aguardar o término de requisições em andamento antes de fechar. Em deploys (ex.: container restart), conexões ativas são encerradas abruptamente — potencial perda de writes em progresso.

**Correção:** Usar `http.Server` com `Shutdown(ctx)` acionado por `signal.NotifyContext`.

---

### 8. Endpoints de listagem sem paginação

**Arquivo:** `internal/http/routes/router.go`

- `GET /v1/events/{eventId}/guests` — retorna **todos** os convidados
- `GET /v1/events/{eventId}/rsvps` — retorna **todos** os RSVPs
- `GET /v1/events/{eventId}/checkin/guests` — retorna **todos** sem filtro

Para eventos com centenas ou milhares de participantes, estas rotas podem gerar respostas de vários MBs, impactando latência, memória e bandwidth.

**Correção:** Adicionar paginação por cursor ou offset/limit, com valores máximos forçados no service.

---

### 9. Token de reset de senha exposto como query parameter

**Arquivo:** `internal/services/auth_password_reset.go` — `buildPasswordResetURL`

```go
query.Set("token", rawToken)
```

O token raw (48 chars aleatórios) é inserido diretamente na query string da URL. URLs aparecem em:
- Logs de acesso do servidor proxy/CDN (header `Referer`)
- Histórico do browser do usuário
- Logs do Brevo (se houver rastreamento de link)

Embora o token seja de uso único e curto prazo (padrão 1h), a exposição aumenta desnecessariamente a janela de comprometimento.

**Correção:** Manter o token no path (`/redefinir-senha/{token}`) em vez de como query param, ou ao menos garantir que os logs de acesso não incluam query strings.

---

### 10. Senha mínima sem requisitos de complexidade

**Arquivo:** `internal/services/auth_service.go` — `validatePassword`

```go
if len(strings.TrimSpace(password)) < 6 {
```

Apenas 8 caracteres são exigidos, sem nenhum requisito de variação (maiúsculas, números, caracteres especiais). Senhas como `"password"` ou `"12345678"` são aceitas.

**Correção:** Pelo menos exigir uma combinação de letras e números, ou implementar verificação contra listas de senhas comuns (ex.: usar biblioteca `zxcvbn`).

---

## P2 — Médios

### 11. Presentes sem expiração de reserva (`reserved` / `pending_payment`)

**Arquivo:** `internal/services/gift_transaction_service.go`

Quando um convidado reserva um presente (`status = "reserved"`) ou registra intenção de Pix (`status = "pending_payment"`), não há TTL. Se o convidado abandonar o processo, o presente fica **bloqueado indefinidamente**, exigindo que o organizador cancele manualmente.

**Correção:** Implementar job periódico (ou trigger SQL) para expirar transações `pending` após N horas e liberar o presente.

---

### 12. Ausência de limite de tamanho nos campos de texto livre

**Arquivos:** DTOs em `internal/dto/`

Campos como `message` (RSVP, gift transaction), `guest_contact`, `notes`, `description` (gift/event), `host_message` não possuem validação de tamanho máximo no service.

**Impacto:** Entradas muito grandes demandam espaço excessivo no banco e aumentam o tamanho de respostas JSON.

**Correção:** Definir e validar limite razoável em cada campo (ex.: `message` ≤ 500 chars, `description` ≤ 2000 chars).

---

### 13. ForgotPassword falha completamente se o serviço de e-mail estiver fora

**Arquivo:** `internal/services/auth_password_reset.go`

```go
if err := s.passwordResetSender.SendPasswordReset(...); err != nil {
    return "", err  // propaga erro para o handler → retorna 500
}
```

Se o Brevo estiver temporariamente indisponível, o token já foi criado no banco, mas o usuário recebe erro 500 e não consegue tentar novamente sem um novo token (o anterior foi consumido de `DeleteActiveByUserID`). O token criado fica "órfão" até expirar.

**Correção:** Logar o erro de envio mas retornar a mensagem de sucesso genérica (o token já está salvo). Ou implementar retry/queue para envio assíncrono.

---

### 14. `MaxCompanions: 10` hardcoded para convidados de OpenRSVP

**Arquivo:** `internal/services/rsvp_service.go:238`

```go
MaxCompanions: 10,
```

Convidados criados automaticamente via RSVP aberto recebem `MaxCompanions = 10` sem qualquer configuração do organizador. O organizador não controla este limite para estes participantes.

**Correção:** Expor configuração de `default_max_companions` no evento para OpenRSVP, ou usar `0` (sem acompanhantes por padrão) como fallback conservador.

---

## P3 — Baixos / Observações

### 15. Healthcheck sem autenticação expõe informações de infraestrutura

**Arquivo:** `internal/http/handlers/health.go`

O endpoint `GET /health` (não protegido) retorna status do banco de dados e do storage. Informações sobre conectividade de infra podem ser úteis para reconhecimento por atacantes.

**Recomendação:** Aceitar como razoável para monitoring interno ou adicionar autenticação por token de infraestrutura.

---

### 16. Logs de erro do Telegram include `user.ID` (PII)

**Arquivo:** `internal/services/auth_service.go:130`

```go
log.Printf("telegram registration notification failed for user %s: %v", user.ID, err)
```

O ID do usuário é dado pessoal nos termos da LGPD. Dependendo do ambiente de log, pode ser persistido sem controle de retenção.

**Recomendação:** Verificar política de retenção de logs. Considerar remover ou mascarar o user ID.

---

### 17. Verificação duplicada de autenticação nos handlers de upload

**Arquivo:** `internal/http/handlers/uploads.go`

Os métodos `Create` e `Delete` verificam `middleware.UserIDFromContext` manualmente, embora ambas as rotas já estejam envolvidas pelo middleware `Authenticator`. A checagem extra é defensiva mas redundante.

**Recomendação:** Remover a verificação manual, pois é garantida pelo middleware de rota.

---

### 18. `_ = filename` — parâmetro lido e descartado

**Arquivo:** `internal/services/upload_service.go`

```go
_ = filename  // o nome original do arquivo é ignorado
```

O filename recebido do multipart é descartado. O comportamento é correto do ponto de vista de segurança (evita path injection via filename), mas o parâmetro ainda circula pela assinatura do método sem uso.

**Recomendação:** Remover o parâmetro `filename` da interface pública do `UploadService.Upload` para deixar a API mais honesta.

---

### 19. Servidor HTTP sem TLS nativo (depende de proxy externo)

**Arquivo:** `cmd/api/main.go`

```go
http.ListenAndServe(":"+cfg.AppPort, router)
```

Não há TLS no próprio servidor. A segurança depende inteiramente do proxy reverso (Nginx, Caddy, Cloudflare). Se o proxy for mal-configurado ou o servidor for exposto diretamente, o tráfego trafega em plain text.

**Recomendação:** Documentar explicitamente o requisito de proxy TLS no README e considerar adicionar checagem que aborte em `production` se `FrontendURL` não usar HTTPS.

---

## Resumo por Prioridade

| # | Prioridade | Item | Arquivos Afetados |
|---|---|---|---|
| 1 | 🔴 P0 | Race condition reserva de presente | `gift_transaction_service.go` |
| 2 | 🔴 P0 | Race condition OpenRSVP create | `rsvp_service.go` |
| 3 | 🔴 P0 | Confirm/Cancel sem transação atômica | `gift_transaction_service.go` |
| 4 | 🔴 P0 | JWT_SECRET com default "super-secret" | `config.go` |
| 5 | 🔴 P0 | Sem rate limiting em rotas públicas | `router.go` |
| 6 | 🟠 P1 | `errors.Is` vs `==` no reset de senha | `auth_password_reset.go` |
| 7 | 🟠 P1 | Sem graceful shutdown | `main.go` |
| 8 | 🟠 P1 | Listagens sem paginação | `router.go`, services |
| 9 | 🟠 P1 | Token de reset exposto em query param | `auth_password_reset.go` |
| 10 | 🟠 P1 | Senha fraca sem complexidade | `auth_service.go` |
| 11 | 🟡 P2 | Reservas sem expiração (TTL) | `gift_transaction_service.go` |
| 12 | 🟡 P2 | Sem limite nos campos de texto | `dto/*.go`, services |
| 13 | 🟡 P2 | ForgotPassword falha se e-mail falhar | `auth_password_reset.go` |
| 14 | 🟡 P2 | MaxCompanions hardcoded no OpenRSVP | `rsvp_service.go` |
| 15 | 🟢 P3 | Healthcheck sem autenticação | `handlers/health.go` |
| 16 | 🟢 P3 | user.ID em logs (PII/LGPD) | `auth_service.go` |
| 17 | 🟢 P3 | Verificação dupla de auth em uploads | `handlers/uploads.go` |
| 18 | 🟢 P3 | Parâmetro filename descartado | `upload_service.go` |
| 19 | 🟢 P3 | Sem TLS nativo no servidor | `main.go` |

---

*Gerado por revisão estática completa do código-fonte. Verificar os pontos P0 antes do próximo deploy em produção.*
