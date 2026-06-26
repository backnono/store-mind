# Customer Chat Session Management Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add customer-side "new chat" and local historical session management so the miniapp does not keep reusing one stale `session_id` across unrelated topics.

**Architecture:** Keep the first implementation frontend-local and API-compatible with the existing chat endpoint. `SessionStore` owns active session and local history metadata; `ChatStore` persists messages/drafts by `store_id + session_id` instead of only by `store_id`; the chat page exposes header actions for starting a new chat and switching history. Backend `SessionID=0` already creates a new session, so no backend change is required for the MVP.

**Tech Stack:** Vue 3, Taro, Pinia, Vitest, existing `/api/v1/customer-qa/chat` endpoint, local Taro storage.

---

## Problem Summary

Current behavior:

- `frontend/src/stores/session.ts` persists one `session_id` per `store_id`.
- `frontend/src/pages/chat/index.vue` restores that saved session on page entry.
- `frontend/src/stores/chat.ts` sends `session_id: sessionStore.currentSessionId || undefined`.
- Message and draft storage keys are only scoped by `store_id`.

This means reopening the chat page keeps using the same server session, such as `session_id=21`. The backend session state can remain `product_focus` with old `context_stack`, causing topic switches like "薯片在哪里？" to be interpreted against stale context.

## MVP Scope

Build:

- A visible `+` action in the chat header for a new chat.
- A history action in the chat header for local historical sessions.
- Local session history metadata per store.
- Active message/draft persistence scoped by session.
- Tests proving new chat clears the active session persistently and the next send omits `session_id`.

Defer:

- Server-side customer session list endpoint.
- Loading full historical messages from backend.
- Deleting or renaming historical sessions.
- Cross-device history sync.

---

### Task 1: Add Session-Scoped Storage Keys

**Files:**

- Modify: `frontend/src/utils/storageKeys.ts`
- Modify: `frontend/src/utils/storage.test.ts`

**Step 1: Write failing tests**

Add tests:

```ts
describe('session scoped keys', () => {
  it('messagesKey supports session scope', () => {
    expect(messagesKey(1, 21)).toBe('store-mind:messages:1:21')
  })

  it('draftKey supports session scope', () => {
    expect(draftKey(1, 21)).toBe('store-mind:draft:1:21')
  })

  it('historyKey contains store_id', () => {
    expect(sessionHistoryKey(1)).toBe('store-mind:session-history:1')
  })
})
```

**Step 2: Verify RED**

Run:

```bash
cd frontend
npm test -- src/utils/storage.test.ts
```

Expected: FAIL because `sessionHistoryKey` is missing and `messagesKey` / `draftKey` accept only one argument.

**Step 3: Implement minimal keys**

Change signatures:

```ts
export function messagesKey(storeId: number, sessionId = 0): string {
  return sessionId > 0
    ? `${PREFIX}:messages:${storeId}:${sessionId}`
    : `${PREFIX}:messages:${storeId}:draft-session`
}

export function draftKey(storeId: number, sessionId = 0): string {
  return sessionId > 0
    ? `${PREFIX}:draft:${storeId}:${sessionId}`
    : `${PREFIX}:draft:${storeId}:draft-session`
}

export function sessionHistoryKey(storeId: number): string {
  return `${PREFIX}:session-history:${storeId}`
}
```

Use `draft-session` for the pre-server session so the first unsent draft does not collide with older store-wide keys.

**Step 4: Verify GREEN**

Run:

```bash
cd frontend
npm test -- src/utils/storage.test.ts
```

Expected: PASS.

---

### Task 2: Fix Session Store New Chat Persistence

**Files:**

- Modify: `frontend/src/stores/session.ts`
- Create: `frontend/src/stores/session.test.ts`

**Step 1: Write failing tests**

Create `session.test.ts`:

```ts
import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import Taro from '@tarojs/taro'
import { useSessionStore } from './session'
import { sessionKey } from '@/utils/storage'

describe('sessionStore', () => {
  beforeEach(async () => {
    setActivePinia(createPinia())
    await Taro.clearStorage()
  })

  it('startNewSession removes active session from memory and storage', async () => {
    const store = useSessionStore()
    store.setStore(1)
    store.setSessionId(21)

    await store.startNewSession()

    expect(store.currentSessionId).toBe(0)
    await expect(Taro.getStorage({ key: sessionKey(1) })).rejects.toThrow()
  })
})
```

**Step 2: Verify RED**

Run:

```bash
cd frontend
npm test -- src/stores/session.test.ts
```

Expected: FAIL because `startNewSession()` is synchronous and does not remove storage.

**Step 3: Implement minimal fix**

In `session.ts`:

- Import `removeItem` from `@/utils/storage`.
- Change `startNewSession()` to `async`.
- Delete the current store id from `sessionIdByStore`.
- Call `await removeItem(sessionKey(storeId.value))`.
- Clear `entryContext`.

Implementation shape:

```ts
async function startNewSession(): Promise<void> {
  if (storeId.value <= 0) return
  const id = storeId.value
  const sids = { ...sessionIdByStore.value }
  delete sids[id]
  sessionIdByStore.value = sids
  entryContext.value = null
  await removeItem(sessionKey(id))
}
```

**Step 4: Verify GREEN**

Run:

```bash
cd frontend
npm test -- src/stores/session.test.ts
```

Expected: PASS.

---

### Task 3: Add Local Session History Metadata

**Files:**

- Modify: `frontend/src/types/customerQa.ts`
- Modify: `frontend/src/stores/session.ts`
- Modify: `frontend/src/stores/session.test.ts`

**Step 1: Write failing tests**

Extend `session.test.ts`:

```ts
it('records sessions in newest-first local history', async () => {
  const store = useSessionStore()
  store.setStore(1)

  await store.recordSession({ sessionId: 21, title: '薯片在哪里？' })
  await store.recordSession({ sessionId: 22, title: '怎么付款？' })

  expect(store.sessionHistory[0].sessionId).toBe(22)
  expect(store.sessionHistory[1].sessionId).toBe(21)
})

it('switchSession updates active session id', async () => {
  const store = useSessionStore()
  store.setStore(1)
  await store.switchSession(21)

  expect(store.currentSessionId).toBe(21)
})
```

**Step 2: Verify RED**

Run:

```bash
cd frontend
npm test -- src/stores/session.test.ts
```

Expected: FAIL because history APIs and type are missing.

**Step 3: Add type**

In `customerQa.ts`:

```ts
export interface LocalSessionSummary {
  sessionId: number
  storeId: number
  title: string
  lastMessagePreview?: string
  updatedAt: number
}
```

**Step 4: Implement session history**

In `session.ts`:

- Add `const sessionHistory = ref<LocalSessionSummary[]>([])`.
- Add `restoreSessionHistory(storeId: number)`.
- Add `recordSession(input: { sessionId: number; title?: string; lastMessagePreview?: string })`.
- Add `switchSession(sessionId: number)`.
- Persist to `sessionHistoryKey(storeId)`.
- Keep newest-first and cap to 20 items.

Title rule:

- If provided, trim and use first 24 characters.
- Else use `会话 ${sessionId}`.

**Step 5: Verify GREEN**

Run:

```bash
cd frontend
npm test -- src/stores/session.test.ts
```

Expected: PASS.

---

### Task 4: Scope Chat Persistence By Active Session

**Files:**

- Modify: `frontend/src/stores/chat.ts`
- Modify: `frontend/src/stores/chat.test.ts`

**Step 1: Write failing test**

Add to `chat.test.ts`:

```ts
it('new chat sends without stale session_id after clearing active session', async () => {
  const sessionStore = useSessionStore()
  const store = useChatStore()
  sessionStore.setSessionId(21)
  await sessionStore.startNewSession()
  store.setDraftText('薯片在哪里？')

  const mockChat = chat as ReturnType<typeof vi.fn>
  mockChat.mockResolvedValue({
    session_id: 22,
    message_id: 88,
    intent: 'product_location',
    answer: '薯片在零食区',
    cards: [],
    guidance_chips: [],
    handoff_required: false,
  })

  await store.sendMessage()

  expect(mockChat).toHaveBeenCalledWith({
    store_id: 1,
    session_id: undefined,
    channel: 'miniapp',
    message: '薯片在哪里？',
  })
  expect(sessionStore.currentSessionId).toBe(22)
})
```

**Step 2: Verify RED**

Run:

```bash
cd frontend
npm test -- src/stores/chat.test.ts
```

Expected: FAIL if `startNewSession` has not been awaited or stale storage/session state still leaks.

**Step 3: Update chat persistence**

In `chat.ts`:

- Use `messagesKey(sessionStore.storeId, sessionStore.currentSessionId)` in `persistMessages()` and `restoreMessages()`.
- Use `draftKey(sessionStore.storeId, sessionStore.currentSessionId)` in draft persistence.
- After a successful chat response, call `sessionStore.recordSession()` with:

```ts
await sessionStore.recordSession({
  sessionId: response.session_id,
  title: text,
  lastMessagePreview: response.answer,
})
```

- After first response assigns a new session id, persist messages again under the new session-scoped key.

**Step 4: Verify GREEN**

Run:

```bash
cd frontend
npm test -- src/stores/chat.test.ts
```

Expected: PASS.

---

### Task 5: Add Header Actions For New Chat And History

**Files:**

- Modify: `frontend/src/components/chat/ChatHeader.vue`
- Modify: `frontend/src/pages/chat/index.vue`
- Modify: `frontend/src/pages/chat/index.scss`

**Step 1: Update header contract**

In `ChatHeader.vue`, replace the single `more` action with two clear icon actions:

- `+` emits `new-chat`.
- `≡` or `☰` emits `history`.

Avoid visible explanatory text in the header; use accessible labels if Taro component support allows it.

Suggested template shape:

```vue
<view class="header-actions">
  <view class="header-btn" aria-label="新聊天" @tap="$emit('new-chat')">+</view>
  <view class="header-btn" aria-label="历史会话" @tap="$emit('history')">≡</view>
</view>
```

Emits:

```ts
defineEmits<{
  'new-chat': []
  history: []
}>()
```

**Step 2: Add page state**

In `index.vue`:

```ts
const showHistory = ref(false)
```

Add handlers:

```ts
async function onNewChat(): Promise<void> {
  await chatStore.persistDraft(sessionStore.storeId)
  await chatStore.persistMessages()
  await sessionStore.startNewSession()
  chatStore.clearChat()
  contextBridge.value = ''
  zoneBanner.value = { show: false, label: '', desc: '' }
  await handleFirstOpen()
}

async function onHistory(): Promise<void> {
  await sessionStore.restoreSessionHistory(sessionStore.storeId)
  showHistory.value = true
}

async function onSelectSession(sessionId: number): Promise<void> {
  await chatStore.persistDraft(sessionStore.storeId)
  await chatStore.persistMessages()
  await sessionStore.switchSession(sessionId)
  chatStore.clearChat()
  await chatStore.restoreMessages(sessionStore.storeId)
  await chatStore.restoreDraft(sessionStore.storeId)
  contextBridge.value = '历史会话'
  showHistory.value = false
}
```

**Step 3: Add minimal history panel**

In `index.vue` template:

```vue
<view v-if="showHistory" class="history-mask" @tap="showHistory = false">
  <view class="history-panel" @tap.stop>
    <view class="history-title">历史会话</view>
    <view
      v-for="item in sessionStore.sessionHistory"
      :key="item.sessionId"
      class="history-item"
      @tap="onSelectSession(item.sessionId)"
    >
      <view class="history-item-title">{{ item.title }}</view>
      <view class="history-item-preview">{{ item.lastMessagePreview || '继续这段对话' }}</view>
    </view>
    <view v-if="sessionStore.sessionHistory.length === 0" class="history-empty">
      暂无历史会话
    </view>
  </view>
</view>
```

**Step 4: Style panel**

In `index.scss`, style as a compact bottom sheet:

- Mask covers the chat page.
- Panel anchors to bottom, max height 70vh.
- Items are dense and readable.
- Do not nest cards inside cards.

**Step 5: Manual verification**

Run:

```bash
cd frontend
npm run dev:weapp
```

In WeChat DevTools:

1. Open chat page.
2. Send "可乐在哪里？".
3. Tap `+`.
4. Send "薯片在哪里？".
5. Confirm backend log uses a new `session_id`.
6. Tap history.
7. Switch back to the older session.
8. Confirm local messages for that session appear without mixing with the new chat.

---

### Task 6: Bootstrap And Resume Behavior Cleanup

**Files:**

- Modify: `frontend/src/pages/chat/index.vue`

**Step 1: Adjust page bootstrap**

Current `bootstrapEntry()` only restores when `entry === 'resume'`. Keep that behavior, but always restore local history metadata on load:

```ts
await sessionStore.restoreSessionHistory(sessionStore.storeId)
```

For `first_open`, do not auto-call backend if there is a saved session. Show welcome state unless explicitly entering via resume or user picks a historical session.

**Step 2: Prevent duplicate welcome messages**

Before `handleFirstOpen()` adds the assistant welcome, guard:

```ts
if (chatStore.messages.length > 0) return
```

**Step 3: Verify manually**

In WeChat DevTools:

1. Open chat page fresh.
2. Confirm it does not silently resume stale `session_id=21`.
3. Send a message and confirm server creates or returns a session.
4. Reload page.
5. Confirm active session is still available only if current product decision wants resume behavior; otherwise `+` can create a clean chat.

---

### Task 7: Full Frontend Verification

**Files:**

- No source edits unless tests reveal issues.

**Step 1: Run unit tests**

Run:

```bash
cd frontend
npm test
```

Expected: PASS.

**Step 2: Run typecheck**

Run:

```bash
cd frontend
npm run typecheck
```

Expected: PASS.

**Step 3: Run build**

Run:

```bash
cd frontend
npm run build:weapp
```

Expected: PASS and `frontend/dist/` updates.

**Step 4: Backend sanity**

No backend change is planned for the MVP. If previous backend changes are still present, run:

```bash
cd backend
make test
```

Expected: PASS.

---

### Task 8: Follow-Up Backend Design Option

**Files:**

- No MVP source edits.
- Future design may touch:
  - `backend/api/http/router.go`
  - `backend/api/http/handler_customer_qa.go`
  - `backend/application/customerqa/service.go`
  - `frontend/src/services/customerQa.ts`

If local history is not enough, add a customer-safe endpoint:

```http
GET /api/v1/customer-qa/sessions?store_id=1&limit=20
```

Return only customer-safe metadata:

```json
{
  "items": [
    {
      "session_id": 22,
      "title": "薯片在哪里？",
      "last_message_preview": "薯片在零食区 A-03",
      "updated_at": "2026-06-23T18:30:00+08:00"
    }
  ]
}
```

Do not expose admin-only tool calls or internal decision logs to the miniapp.

---

## Acceptance Criteria

- Tapping `+` clears the active local session and storage-backed active session id.
- The next real user message after `+` sends `session_id: undefined`.
- Backend creates a new session for that next message.
- History panel shows local sessions newest-first.
- Switching history changes active `session_id`.
- Messages and drafts do not mix between sessions.
- Reopening the page no longer forces all unrelated topics into one old session.
- Existing feedback submission still uses the active session id.

## Rollback Plan

If the UI causes issues:

1. Hide the history button and panel.
2. Keep the `+` new chat path, because it directly fixes stale context.
3. If necessary, disable local history writes while preserving session-scoped message keys.

