# Customer Miniapp Architecture Design

## Goal

Build the first-phase customer-facing "小王" miniapp experience with Taro, Vue 3, TypeScript, and a WeChat mini program build target. This phase focuses on reliable customer Q&A, not operations admin screens.

## Scope

In scope:

- Store card home page after a customer enters the store.
- Unified "小王" chat page.
- Entry adaptation for first open, zone shelf scan, promotion/recommendation entry, and resumed sessions.
- Product, inventory, promotion, FAQ, price, and fallback answer rendering.
- Guidance chips that fill the input box without sending automatically.
- Per-assistant-message thumbs up/down feedback.
- Local session persistence by store.

Out of scope:

- Operations/admin CRUD UI.
- Login, membership, payment, cart, refund workflow, and checkout.
- H5/App targets. The first build target is WeChat mini program only.
- Voice input/output.

## Architecture

Use a single Taro project under `frontend/`, not a frontend monorepo. The project structure is intentionally small but not throwaway:

```text
frontend/
  package.json
  project.config.json
  config/
  src/
    app.config.ts
    app.ts
    app.scss
    pages/
      store/
        index.vue
        index.config.ts
        index.scss
      chat/
        index.vue
        index.config.ts
        index.scss
    components/
      store/
      chat/
      common/
    services/
      api.ts
      customerQa.ts
    stores/
      session.ts
      chat.ts
    types/
      customerQa.ts
    utils/
      env.ts
      qr.ts
      storage.ts
```

Pages are thin shells for lifecycle, routing, and user events. Pinia stores own session and chat state. Services own HTTP transport and DTO mapping. Components are presentation-first and receive explicit props.

## Pages

### Store Page

`pages/store/index` is the customer entry surface. It shows store identity, 小王's greeting, online state, today's recommendations, and two CTAs:

- "问小王": navigate to `pages/chat/index?entry=first_open&store_id=...`.
- "扫码购物": call `Taro.scanCode`, parse the result, then navigate to `entry=zone_scan`.

Recommendation cards navigate to chat with `entry=promo` and a prompt/context payload. In phase one, recommendations can be static or fetched from active promotions. They should feel like an entry into a conversation, not a marketing feed.

### Chat Page

`pages/chat/index` is the only conversation surface. Query parameters choose the initial mode:

- `first_open`: request the backend greeting and preset guidance chips.
- `zone_scan`: pass `zone_id` and `shelf_id`, then render the zone banner and shelf product cards from the chat response.
- `promo`: bring recommendation context into the conversation.
- `resume`: continue a stored `session_id` and let the backend provide the context bridge.

After bootstrap, all normal user messages use the same `sendMessage` flow.

## Components

Store components:

- `StoreGreeting.vue`: 小王 avatar, status, greeting copy, CTAs.
- `StoreRecommendationList.vue`: promotion/new/hot recommendations.

Chat components:

- `ChatHeader.vue`: 小王 identity and status.
- `ZoneBanner.vue`: current zone/shelf context.
- `MessageList.vue`: scrollable conversation list.
- `MessageBubble.vue`: user and assistant text bubbles.
- `AnswerCard.vue`: renders card variants by `card.type`.
- `GuidanceChips.vue`: emits selected prompt text without sending.
- `FeedbackBar.vue`: handles thumbs up/down state.
- `ChatInput.vue`: draft input, send action, retry affordance.

Common components:

- `StatusPill.vue`
- `InlineNotice.vue`
- `EmptyState.vue`

## State

Use Pinia.

`sessionStore` owns:

- `storeId`
- `sessionIdByStore`
- `entryContext`
- `lastActiveAt`

`chatStore` owns:

- `messages`
- `draftText`
- `isSending`
- `bootstrapStatus`
- `lastError`
- `feedbackByMessageId`

Persist only the useful recovery state: `session_id`, last entry context, and the latest 20 message snapshots. Storage keys must include `store_id`, for example `store-mind:session:1`, so different stores never share a session by accident.

## API Flow

The frontend treats `/api/v1/customer-qa/chat` as the main customer experience endpoint. It does not locally decide whether a question is inventory, FAQ, or promotion. The backend returns `intent`, `answer`, `cards`, `guidance_chips`, `handoff_required`, and `meta`; the frontend maps that response to view models.

Chat request shape:

```ts
type ChatRequest = {
  store_id: number
  session_id?: number
  user_id?: number
  channel: 'miniapp'
  message: string
  entry_mode?: 'first_open' | 'zone_scan' | 'resume' | 'promo' | 'product_detail'
  zone_id?: number
  shelf_id?: number
}
```

Chat send flow:

```text
User input
 -> chatStore.sendMessage()
 -> append local user message
 -> customerQaApi.chat()
 -> append assistant message
 -> render answer/cards/guidance/feedback
 -> persist session_id by store
```

Feedback flow:

```text
FeedbackBar click
 -> customerQaApi.submitFeedback(message_id, session_id, feedback_value)
 -> mark feedback locally
```

Errors are normalized in the API layer:

```ts
type ApiError = {
  code: 'network' | 'timeout' | 'bad_request' | 'server_error' | 'unknown'
  message: string
  retryable: boolean
}
```

UI copy should be calm and user-facing. Do not expose raw backend errors.

## QR Parsing

Phase one supports:

```text
storemind://zone?store_id=1&zone_id=2&shelf_id=5
https://.../miniapp/chat?store_id=1&zone_id=2&shelf_id=5
```

Successful parsing navigates to:

```text
pages/chat/index?entry=zone_scan&store_id=1&zone_id=2&shelf_id=5
```

Failed parsing stays on the store page and shows a short notice.

## UX Rules

- 小王 is proactive but not pushy.
- Guidance chips fill the input box only; users decide whether to send.
- Inventory confidence must be shown with both color and text when available.
- Assistant answers can have feedback. User messages cannot.
- Loading states should keep the conversation usable: show the user's sent message immediately, then show assistant pending state.
- No generic "AI assistant" tone.

## Testing Strategy

Unit tests:

- API mapper.
- QR parser.
- Storage key and recovery helpers.
- Chat store message reducer.
- Feedback state transitions.

Component tests:

- `AnswerCard` renders product, inventory, promotion, price, and fallback cards.
- `GuidanceChips` emits prompt text without sending.
- `FeedbackBar` submits once and locks selected state.

Manual WeChat DevTools acceptance:

- Store page "问小王" opens `first_open`.
- QR scan opens `zone_scan` with banner and cards.
- "可乐在哪" renders assistant answer and cards.
- Guidance chip fills the input without auto-send.
- Feedback submits and remains selected.
- Network/server failure shows retryable notice.

## Open Follow-Ups

- Confirm whether recommendations are static seed data or fetched from `/api/v1/customer-qa/promotions/active` in the first implementation slice.
- Confirm local development API base URL and WeChat DevTools proxy strategy.
- Confirm whether the first implementation should include visual parity screenshots against `docs/design/prototypes/v3.html`.
