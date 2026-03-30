# 📦 MyEvent — Escopo do Produto (MVP)

## 🧭 Visão do Produto

O **MyEvent** é uma plataforma para criação de sites de eventos com gestão de convidados, confirmação de presença (RSVP), check-in via QR code e lista de presentes com pagamento via Pix manual (sem taxa).

A proposta é permitir que qualquer pessoa crie e gerencie um evento completo em poucos minutos, através de um único link.

---

## 🎯 Objetivo Principal

Permitir que usuários:

* criem um site de evento personalizado
* compartilhem com convidados
* gerenciem confirmações de presença
* validem entrada via QR code
* disponibilizem lista de presentes sem taxa

---

## ⚠️ Problemas que Resolve

* Falta de organização de convidados (uso de planilhas/WhatsApp)
* Dificuldade em controlar confirmações de presença
* Ausência de ferramentas simples para check-in no evento
* Plataformas de presentes com taxas altas
* Criação de sites complexa ou limitada

---

## 👤 Público-Alvo

* Casais (casamento)
* Eventos familiares (chá, aniversário)
* Organizadores de eventos pequenos
* Usuários que querem simplicidade e baixo custo

---

## 💎 Proposta de Valor

* Tudo em um único link
* Fácil de criar e compartilhar
* RSVP simples
* Check-in com QR code
* Lista de presentes sem taxa (Pix direto)
* Interface mobile-first

---

## 🧱 Stack Técnica

* **Front-end:** Next.js (App Router) + TailwindCSS
* **Back-end:** Go (API REST)
* **Banco de Dados:** PostgreSQL
* **Storage:** Cloudflare R2

---

## 🧩 Estrutura do Sistema

O sistema é dividido em 3 partes:

1. **Painel do Organizador**
2. **Site Público do Evento**
3. **Subapp de Check-in (QR Code)**

---

## 🚀 MVP — Funcionalidades

### 🔐 Autenticação

* Cadastro com email
* Login
* Recuperação de senha

---

### 📅 Eventos

* Criar evento
* Editar evento
* Definir:

  * nome
  * tipo (casamento, aniversário, etc)
  * data
  * hora
  * local
  * descrição
* Definir slug (URL)
* Publicar / despublicar

---

### 🌐 Site Público

* Página acessível via `/e/[slug]`
* Exibe:

  * nome do evento
  * capa
  * descrição
  * data e hora
  * local
  * mapa/endereço
  * mensagem do anfitrião
  * galeria de fotos
  * seção de presentes
  * botão RSVP

---

### 🎨 Aparência (Temas)

* Seleção de tema:

  * classic
  * minimal
  * party
* Customização:

  * cor primária
  * cor secundária
  * cor de fundo
  * cor de texto
* Renderização via CSS variables
* Preview em tempo real

---

### 👥 Convidados

* Cadastro manual
* Campos:

  * nome
  * email (opcional)
  * telefone (opcional)
  * limite de acompanhantes
* Status:

  * pendente
  * confirmado
  * recusado

---

### ✅ RSVP

* Acesso via link público
* Confirmação de presença
* Definição de acompanhantes
* Mensagem opcional
* Atualização em tempo real no painel

---

### 🔳 QR Code

* QR code único por convidado
* QR code geral do evento
* Validação no check-in

---

### 🎟️ Check-in (Subapp)

* Leitura de QR code
* Busca manual de convidado
* Marcar presença
* Evitar duplicidade de entrada

---

### 🎁 Lista de Presentes (Pix Manual)

#### Funcionalidades

* Criar presentes:

  * nome
  * descrição
  * valor (opcional)
  * imagem
  * link externo (opcional)
* Ações do convidado:

  * reservar presente
  * pagar via Pix
  * marcar como "já paguei"

---

#### Fluxo Pix Manual

* Exibir:

  * chave Pix
  * nome do recebedor
  * valor
* Pagamento feito fora da plataforma
* Usuário informa pagamento
* Organizador confirma manualmente

---

#### Status do Presente

* available
* reserved
* pending_payment
* confirmed

---

#### Dados do Evento (Pix)

* pix_key
* pix_holder_name

---

### 📊 Painel do Organizador

* Visualizar:

  * total de convidados
  * confirmados
  * pendentes
* Gerenciar:

  * convidados
  * presentes
  * confirmações
  * check-in

---

## 🔁 Fluxos Principais

### 1. Criação de Evento

* usuário cria conta
* cria evento
* define slug
* publica

---

### 2. RSVP

* convidado acessa link
* confirma presença
* define acompanhantes

---

### 3. Check-in

* leitura de QR
* valida convidado
* marca presença

---

### 4. Presentes

* convidado seleciona presente
* reserva ou paga via Pix
* organizador confirma manualmente

---

## 📏 Regras de Negócio

* slug do evento deve ser único
* evento pode estar:

  * draft
  * published
  * closed
* RSVP só funciona se evento estiver ativo
* convidado respeita limite de acompanhantes
* QR é único por convidado
* check-in não pode duplicar sem alerta
* presente reservado não aparece disponível
* pagamento Pix é externo e manual

---

## 🧬 Modelagem de Dados (Resumo)

### User

* id
* name
* email
* password_hash

---

### Event

* id
* user_id
* title
* slug
* type
* description
* date
* time
* location
* cover_image
* theme
* primary_color
* secondary_color
* background_color
* text_color
* pix_key
* pix_holder_name
* status

---

### Guest

* id
* event_id
* name
* email
* phone
* invite_code
* qr_code
* max_companions
* rsvp_status
* checked_in_at

---

### RSVP

* id
* guest_id
* companions_count
* message
* status

---

### Gift

* id
* event_id
* title
* description
* image_url
* value
* external_link
* status

---

### GiftTransaction

* id
* gift_id
* event_id
* guest_name
* type (reservation | pix)
* status (pending | confirmed)
* message
* created_at
* confirmed_at

---

## 🚫 Fora do MVP

* pagamento automático (Stripe/Mercado Pago)
* automação de WhatsApp
* domínio customizado
* editor visual avançado
* multiusuário com permissões
* relatórios avançados
* app mobile nativo

---

## 💰 Modelo de Monetização (Inicial)

Plano Free:

* 1 evento
* funcionalidades básicas

Plano Pro:

* convidados ilimitados
* QR check-in
* temas premium
* lista de presentes completa

---

## 🔥 Posicionamento

> “Crie o site do seu evento e gerencie convidados, confirmações e presentes em um só lugar.”

Ou:

> “Seu evento com site, RSVP e check-in por QR code — sem taxa nos presentes.”

---

## ✅ Escopo MVP Final

* autenticação
* criação de evento
* site público com slug
* temas + cores
* convidados
* RSVP
* QR code
* check-in
* presentes com Pix manual

---
