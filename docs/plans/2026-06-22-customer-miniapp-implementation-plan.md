# Customer Miniapp Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the first-phase customer-facing "小王" WeChat mini program experience with Taro, Vue 3, TypeScript, and Pinia.

**Architecture:** Create a single Taro project under `frontend/` with two pages: store home and chat. Keep pages thin, move conversation/session state into Pinia, isolate backend calls in `services/`, and render backend chat responses through typed chat components.

**Tech Stack:** Taro, Vue 3, TypeScript, Pinia, WeChat mini program target, Vitest where practical for pure utilities and stores.

---

### Task 1: Scaffold Taro Vue TypeScript Project

**Files:**

- Create: `frontend/package.json`
- Create: `frontend/project.config.json`
- Create: `frontend/config/index.ts`
- Create: `frontend/config/dev.ts`
- Create: `frontend/config/prod.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/src/app.ts`
- Create: `frontend/src/app.config.ts`
- Create: `frontend/src/app.scss`

**Step 1: Create the package manifest**

Add scripts:

```json
{
  "scripts": {
    "dev:weapp": "taro build --type weapp --watch",
    "build:weapp": "taro build --type weapp",
    "test": "vitest run",
    "typecheck": "vue-tsc --noEmit"
  }
}
```

Include Taro Vue dependencies, Vue 3, Pinia, TypeScript, Sass, Vitest, and Vue type checking.

**Step 2: Create Taro config files**

Configure the app name, source root `src`, output root `dist`, Vue framework, and WeChat mini program target.

**Step 3: Create app shell**

Register Pinia in `src/app.ts` and declare pages in `src/app.config.ts`:

```ts
export default defineAppConfig({
  pages: ['pages/store/index', 'pages/chat/index'],
  window: {
    navigationStyle: 'custom',
    backgroundColor: '#191816',
  },
})
```

**Step 4: Verify**

Run from `frontend/`:

```bash
npm install
npm run typecheck
npm run build:weapp
```

Expected: dependencies install, TypeScript passes, and WeChat mini program output is generated under `frontend/dist`.

**Step 5: Commit**

```bash
git add frontend/package.json frontend/project.config.json frontend/config frontend/tsconfig.json frontend/src/app.ts frontend/src/app.config.ts frontend/src/app.scss
git commit -m "feat: scaffold customer miniapp"
```

### Task 2: Add Shared Types, API Client, and Utilities

**Files:**

- Create: `frontend/src/types/customerQa.ts`
- Create: `frontend/src/services/api.ts`
- Create: `frontend/src/services/customerQa.ts`
- Create: `frontend/src/utils/env.ts`
- Create: `frontend/src/utils/storage.ts`
- Create: `frontend/src/utils/qr.ts`
- Create: `frontend/src/utils/qr.test.ts`
- Create: `frontend/src/utils/storage.test.ts`

**Step 1: Write failing tests for QR parsing and storage keys**

Test both supported QR formats:

```text
storemind://zone?store_id=1&zone_id=2&shelf_id=5
https://example.com/miniapp/chat?store_id=1&zone_id=2&shelf_id=5
```

Expected parsed result:

```ts
{ storeId: 1, zoneId: 2, shelfId: 5 }
```

Test storage keys include `store_id`:

```ts
sessionKey(1) === 'store-mind:session:1'
```

**Step 2: Run tests and confirm failure**

```bash
npm test -- src/utils/qr.test.ts src/utils/storage.test.ts
```

Expected: fail because utilities do not exist yet.

**Step 3: Implement utilities and API client**

`services/api.ts` wraps `Taro.request` and normalizes errors:

```ts
export type ApiError = {
  code: 'network' | 'timeout' | 'bad_request' | 'server_error' | 'unknown'
  message: string
  retryable: boolean
}
```

`services/customerQa.ts` exposes:

- `chat(request: ChatRequest): Promise<ChatResponse>`
- `submitFeedback(request: FeedbackRequest): Promise<void>`
- `listActivePromotions(storeId: number): Promise<Promotion[]>`

**Step 4: Run tests**

```bash
npm test -- src/utils/qr.test.ts src/utils/storage.test.ts
npm run typecheck
```

Expected: tests and typecheck pass.

**Step 5: Commit**

```bash
git add frontend/src/types frontend/src/services frontend/src/utils
git commit -m "feat: add customer qa frontend API utilities"
```

### Task 3: Implement Pinia Session and Chat Stores

**Files:**

- Create: `frontend/src/stores/session.ts`
- Create: `frontend/src/stores/chat.ts`
- Create: `frontend/src/stores/chat.test.ts`
- Modify: `frontend/src/types/customerQa.ts`

**Step 1: Write failing store tests**

Cover:

- `session_id` persists by `store_id`.
- Sending a message appends a local user message before the API resolves.
- Successful response appends assistant message and saves `session_id`.
- Failed send marks the user message as failed and exposes retryable error.
- Feedback can only be applied to assistant messages with `messageId`.

**Step 2: Run tests and confirm failure**

```bash
npm test -- src/stores/chat.test.ts
```

Expected: fail because stores do not exist yet.

**Step 3: Implement stores**

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

Persist latest 20 messages and session ids with Taro storage helpers.

**Step 4: Run tests**

```bash
npm test -- src/stores/chat.test.ts
npm run typecheck
```

Expected: tests and typecheck pass.

**Step 5: Commit**

```bash
git add frontend/src/stores frontend/src/types/customerQa.ts
git commit -m "feat: add customer miniapp chat state"
```

### Task 4: Build Store Page

**Files:**

- Create: `frontend/src/pages/store/index.vue`
- Create: `frontend/src/pages/store/index.config.ts`
- Create: `frontend/src/pages/store/index.scss`
- Create: `frontend/src/components/store/StoreGreeting.vue`
- Create: `frontend/src/components/store/StoreRecommendationList.vue`
- Create: `frontend/src/components/common/InlineNotice.vue`
- Modify: `frontend/src/utils/qr.ts`

**Step 1: Implement store page layout**

Render:

- Store name and address.
- 小王 greeting and online status.
- "问小王" CTA.
- "扫码购物" CTA.
- Today's recommendation list.

**Step 2: Implement navigation**

- "问小王" navigates to `/pages/chat/index?entry=first_open&store_id=1`.
- Recommendation navigates to `/pages/chat/index?entry=promo&store_id=1&prompt=...`.
- Scan success navigates to `/pages/chat/index?entry=zone_scan&store_id=...&zone_id=...&shelf_id=...`.
- Scan parse failure shows `InlineNotice`.

**Step 3: Run verification**

```bash
npm run typecheck
npm run build:weapp
```

Expected: page compiles into WeChat mini program output.

**Step 4: Commit**

```bash
git add frontend/src/pages/store frontend/src/components/store frontend/src/components/common frontend/src/utils/qr.ts
git commit -m "feat: add customer store entry page"
```

### Task 5: Build Chat Components

**Files:**

- Create: `frontend/src/components/chat/ChatHeader.vue`
- Create: `frontend/src/components/chat/ZoneBanner.vue`
- Create: `frontend/src/components/chat/MessageList.vue`
- Create: `frontend/src/components/chat/MessageBubble.vue`
- Create: `frontend/src/components/chat/AnswerCard.vue`
- Create: `frontend/src/components/chat/GuidanceChips.vue`
- Create: `frontend/src/components/chat/FeedbackBar.vue`
- Create: `frontend/src/components/chat/ChatInput.vue`
- Create: `frontend/src/components/chat/GuidanceChips.test.ts`
- Create: `frontend/src/components/chat/AnswerCard.test.ts`

**Step 1: Write failing component tests**

Cover:

- `GuidanceChips` emits selected prompt and does not emit send.
- `AnswerCard` renders at least product, inventory, promotion, price, and fallback card variants.

**Step 2: Run tests and confirm failure**

```bash
npm test -- src/components/chat/GuidanceChips.test.ts src/components/chat/AnswerCard.test.ts
```

Expected: fail because components do not exist yet.

**Step 3: Implement components**

Keep components presentation-first. No direct backend calls except `FeedbackBar`, which emits an event to the page/store rather than calling the service itself.

**Step 4: Run tests**

```bash
npm test -- src/components/chat/GuidanceChips.test.ts src/components/chat/AnswerCard.test.ts
npm run typecheck
```

Expected: tests and typecheck pass.

**Step 5: Commit**

```bash
git add frontend/src/components/chat
git commit -m "feat: add chat presentation components"
```

### Task 6: Build Chat Page

**Files:**

- Create: `frontend/src/pages/chat/index.vue`
- Create: `frontend/src/pages/chat/index.config.ts`
- Create: `frontend/src/pages/chat/index.scss`
- Modify: `frontend/src/stores/session.ts`
- Modify: `frontend/src/stores/chat.ts`

**Step 1: Implement bootstrap from query**

Read:

- `entry`
- `store_id`
- `zone_id`
- `shelf_id`
- `prompt`

Call `chatStore.bootstrapEntry()` for `first_open`, `zone_scan`, `promo`, and `resume`.

**Step 2: Implement normal send flow**

Wire `ChatInput` to `chatStore.sendMessage()`. Guidance chip clicks set `draftText` only.

**Step 3: Implement feedback flow**

Wire `FeedbackBar` to `chatStore.submitFeedback(messageId, value)`.

**Step 4: Implement scroll behavior and loading states**

Scroll to bottom after new messages. Show pending assistant state while waiting. Show retry affordance on failed user message.

**Step 5: Run verification**

```bash
npm run typecheck
npm run build:weapp
```

Expected: chat page compiles and routes are included in app config.

**Step 6: Commit**

```bash
git add frontend/src/pages/chat frontend/src/stores
git commit -m "feat: add customer chat page"
```

### Task 7: Visual Polish and Design Parity

**Files:**

- Modify: `frontend/src/app.scss`
- Modify: `frontend/src/pages/store/index.scss`
- Modify: `frontend/src/pages/chat/index.scss`
- Modify: `frontend/src/components/**/*.vue`

**Step 1: Apply design tokens**

Use the dark, warm, restrained product palette from `docs/design/design.md`. Keep typography system-native and readable on mobile.

**Step 2: Match the prototype structure**

Compare against `docs/design/prototypes/v3.html`:

- Store greeting prominence.
- Chat header.
- Zone banner.
- Message bubbles.
- Product/inventory/promotion cards.
- Guidance chips.
- Feedback controls.

**Step 3: Run build**

```bash
npm run typecheck
npm run build:weapp
```

Expected: no type or build failures.

**Step 4: Manual WeChat DevTools check**

Open `frontend/dist` in WeChat DevTools and verify:

- Store page loads.
- "问小王" opens chat first-open mode.
- Chat layout fits a mobile viewport.
- Input and chips do not overlap bottom safe area.

**Step 5: Commit**

```bash
git add frontend/src
git commit -m "style: polish customer miniapp experience"
```

### Task 8: Documentation and Acceptance Checklist

**Files:**

- Create: `frontend/README.md`
- Create: `docs/testing/customer-miniapp-verification.md`
- Modify: `.gitignore`

**Step 1: Document local setup**

Include:

- `npm install`
- `npm run dev:weapp`
- `npm run build:weapp`
- API base URL configuration.
- How to open `frontend/dist` in WeChat DevTools.

**Step 2: Document manual acceptance**

Checklist:

- Store page first-open.
- QR zone scan.
- Product location question.
- Guidance chip fills input only.
- Feedback submission.
- Network/server error state.

**Step 3: Update `.gitignore`**

Ignore generated frontend artifacts such as:

```text
frontend/node_modules/
frontend/dist/
```

**Step 4: Final verification**

```bash
cd frontend
npm test
npm run typecheck
npm run build:weapp
cd ../backend
python3 scripts/validate.py
```

Expected: frontend tests/typecheck/build pass, backend validation still passes.

**Step 5: Commit**

```bash
git add frontend/README.md docs/testing/customer-miniapp-verification.md .gitignore
git commit -m "docs: add customer miniapp verification guide"
```
