# Chat Prototype Scroll Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate the `prototype-a-v3.html` chat scrolling experience into the WeChat miniapp chat page so users can swipe up to view history, while the latest user-initiated exchange stays visible above the composer and the composer is not covered by the custom tabbar.

**Architecture:** Move scroll ownership to the chat page, matching the prototype's single `chat-area` model. `MessageList` becomes a pure content renderer, while `frontend/src/pages/chat/index.vue` owns the `scroll-view`, bottom anchor, near-bottom state, and programmatic stick-to-bottom behavior. Add a `composer-spacer` inside the scroll content so the last message can scroll above the overlaid composer.

**Tech Stack:** Taro 4, Vue 3 Composition API, WeChat miniapp `scroll-view`, Pinia chat/session stores, Vitest node-based source contract tests.

---

## Context

The HTML prototype works because its layout is a normal browser flex stack:

```text
phone-frame
├─ header
├─ chat-area        ← flex:1, native overflow-y:auto
├─ input-area       ← normal layout item
└─ tab-bar          ← normal layout item
```

The WeChat miniapp differs in two important ways:

- The custom tabbar visually covers the bottom of the page and does not behave like a normal page flex child.
- `scroll-view` is controlled by declarative props such as `scroll-into-view` and has async rendering behavior, unlike direct DOM `scrollTop = scrollHeight`.

Current failure mode:

- Reserving enough bottom space prevents the composer from being covered by the tabbar.
- But messages still become hidden behind the composer because the scroll content does not reserve composer height.
- Attempts to control scrolling inside `MessageList` make history scrolling fragile because the content component and page both influence scroll behavior.

The migration should introduce a miniapp-specific spacer:

```text
chat-page
├─ ChatHeader
├─ ZoneBanner
├─ chat-body
│  ├─ scroll-view.chat-scroll
│  │  └─ messages-inner
│  │     ├─ MessageList
│  │     ├─ composer-spacer
│  │     └─ bottom-anchor
│  └─ ChatInput       ← overlaid above custom tabbar
└─ custom tabbar reserve
```

---

## Task 1: Lock The New Scroll Ownership Contract

**Files:**
- Modify: `frontend/src/pages/chat/index-layout.test.ts`
- Modify: `frontend/src/components/chat/MessageList.test.ts`

**Step 1: Replace the chat page layout test**

Update `frontend/src/pages/chat/index-layout.test.ts` so it expects:

- `index.vue` contains `<scroll-view`.
- The scroll-view has `class="chat-scroll-area"`.
- The scroll-view has `scroll-y`.
- The scroll-view has `:scroll-into-view="scrollIntoView"`.
- The scroll-view has `@scroll="handleMessageScroll"`.
- The scroll-view contains a composer spacer and bottom anchor.
- `ChatInput` lives in `class="chat-composer-slot"`.
- `index.scss` contains `--chat-tabbar-reserve`.
- `index.scss` contains `--chat-composer-height`.
- `.chat-body` is `position: relative`.
- `.chat-scroll-area` is the only message scroll surface.
- `.chat-composer-slot` is positioned above the tabbar reserve.
- `.composer-spacer` has height tied to `--chat-composer-height`.

Use source-string tests similar to the existing file; do not introduce jsdom for this contract.

**Step 2: Replace the MessageList test**

Update `frontend/src/components/chat/MessageList.test.ts` so it expects `MessageList.vue`:

- Does not contain `<scroll-view`.
- Does not contain `scroll-y`.
- Does not contain `scrollTop`.
- Does not contain `scrollIntoView`.
- Does not contain `viewportHeight`.
- Does not contain `stickToBottomRequest`.
- Does render `<view class="messages-inner">`.
- Does render `MessageBubble`.
- Does render `messages-bottom-spacer`.

**Step 3: Run tests and verify they fail**

Run:

```bash
cd frontend
npm test -- src/pages/chat/index-layout.test.ts src/components/chat/MessageList.test.ts src/components/chat/ChatInput.test.ts
```

Expected:

- `index-layout.test.ts` fails because the page has not yet moved scroll ownership to `index.vue`.
- `MessageList.test.ts` fails if `MessageList.vue` still owns `scroll-view` or scroll state.

---

## Task 2: Make MessageList A Pure Content Renderer

**Files:**
- Modify: `frontend/src/components/chat/MessageList.vue`
- Test: `frontend/src/components/chat/MessageList.test.ts`

**Step 1: Remove the owned scroll-view**

Change the template from:

```vue
<scroll-view ...>
  <view class="messages-inner">...</view>
</scroll-view>
```

to:

```vue
<template>
  <view class="messages-inner">
    <view v-if="contextBridge" class="context-bridge">
      您之前在看 <text class="cb-hl">{{ contextBridge }}</text>，需要继续吗？
    </view>

    <view v-for="msg in filteredMessages" :key="msg.id">
      <MessageBubble
        :text="msg.text"
        :role="msg.role"
        :message-id="msg.messageId"
        :cards="msg.cards"
        :chips="msg.guidanceChips"
        :feedback-value="getFeedback(msg.messageId)"
        :send-status="msg.sendStatus"
        @chip-select="(p: string) => $emit('chipSelect', p)"
        @feedback-submit="(mid: number, v: 0 | 1) => $emit('feedbackSubmit', mid, v)"
        @retry="$emit('retry', msg.id)"
      />
    </view>

    <view v-if="isThinking" class="typing-row">
      <view class="msg-avatar clerk-av">王</view>
      <view class="typing-content">
        <view class="ai-name">王</view>
        <view class="typing-dots">
          <view class="dot"></view>
          <view class="dot"></view>
          <view class="dot"></view>
        </view>
      </view>
    </view>

    <view class="messages-bottom-spacer"></view>
  </view>
</template>
```

**Step 2: Remove scroll props and state**

Keep only these props:

```ts
const props = defineProps<{
  messages: Message[]
  isThinking: boolean
  contextBridge?: string
}>()
```

Remove:

- `viewportHeight`
- `stickToBottomRequest`
- `scrollIntoView`
- `scrollTop`
- `scrollTopAnimated`
- `autoScrollEnabled`
- scroll watchers
- `handleScroll`
- `scrollToBottom`

**Step 3: Keep only content styles**

Remove `.messages { ... }` from `MessageList.vue`.

Keep:

```scss
.messages-inner {
  box-sizing: border-box;
  min-height: 100%;
  padding: 24px 20px 0;
}

.messages-bottom-spacer {
  height: 16px;
  flex-shrink: 0;
}
```

**Step 4: Run focused tests**

Run:

```bash
cd frontend
npm test -- src/components/chat/MessageList.test.ts
```

Expected: pass.

---

## Task 3: Move Scroll Ownership To The Chat Page

**Files:**
- Modify: `frontend/src/pages/chat/index.vue`
- Modify: `frontend/src/pages/chat/index.scss`
- Test: `frontend/src/pages/chat/index-layout.test.ts`

**Step 1: Replace the message panel with page-owned scroll-view**

In `frontend/src/pages/chat/index.vue`, replace the message panel wrapper with:

```vue
<view class="chat-body">
  <scroll-view
    class="chat-scroll-area"
    scroll-y
    :scroll-into-view="scrollIntoView"
    :scroll-with-animation="scrollWithAnimation"
    @scroll="handleMessageScroll"
    :enable-flex="true"
    :bounces="false"
    :enable-back-to-top="false"
    :enhanced="true"
    :show-scrollbar="true"
  >
    <MessageList
      :messages="chatStore.messages"
      :is-thinking="isThinking"
      :context-bridge="contextBridge"
      @chip-select="onChipSelect"
      @feedback-submit="onFeedbackSubmit"
      @retry="onRetry"
    />
    <view class="composer-spacer"></view>
    <view :id="bottomAnchorId" class="chat-scroll-bottom-anchor"></view>
  </scroll-view>

  <view class="chat-composer-slot">
    <ChatInput
      v-model="chatStore.draftText"
      :disabled="chatStore.isSending"
      placeholder="问小王任何问题…"
      @send="onSend"
    />
  </view>
</view>
```

**Step 2: Add scroll state to page script**

Add:

```ts
const messageViewportHeight = ref(0)
const scrollIntoView = ref('')
const scrollWithAnimation = ref(true)
const autoStickToBottom = ref(true)
const bottomAnchorSerial = ref(0)
const bottomAnchorId = computed(() => `chat-scroll-bottom-${bottomAnchorSerial.value}`)
let releaseScrollTargetTimer: ReturnType<typeof setTimeout> | undefined

type SelectorRect = {
  height?: number
}

type ScrollDetail = {
  scrollTop?: number
  scrollHeight?: number
}

type ScrollEvent = {
  detail?: ScrollDetail
}

const AUTO_SCROLL_THRESHOLD_PX = 80
```

Ensure `computed` is imported from `vue`.

**Step 3: Add page-level scroll helpers**

Add:

```ts
async function updateMessageViewportHeight(): Promise<void> {
  await nextTick()
  Taro.createSelectorQuery()
    .select('.chat-scroll-area')
    .boundingClientRect((rect: SelectorRect | SelectorRect[] | null) => {
      const box = Array.isArray(rect) ? rect[0] : rect
      if (box?.height && box.height > 0) {
        messageViewportHeight.value = Math.floor(box.height)
      }
    })
    .exec()
}

function isNearBottom(detail: ScrollDetail): boolean {
  const scrollTop = detail.scrollTop ?? 0
  const scrollHeight = detail.scrollHeight ?? 0
  const viewportHeight = messageViewportHeight.value

  if (!scrollHeight || !viewportHeight) return true

  return (scrollHeight - (scrollTop + viewportHeight)) <= AUTO_SCROLL_THRESHOLD_PX
}

function handleMessageScroll(event: ScrollEvent): void {
  autoStickToBottom.value = isNearBottom(event.detail ?? {})
}

function requestStickToBottom(animated = true): void {
  void scrollToBottom(true, animated)
}

async function scrollToBottom(force = false, animated = true): Promise<void> {
  if (!force && !autoStickToBottom.value) return
  await nextTick()
  bottomAnchorSerial.value += 1
  await nextTick()
  scrollWithAnimation.value = animated
  scrollIntoView.value = bottomAnchorId.value
  releaseScrollControl()
}

function releaseScrollControl(): void {
  if (releaseScrollTargetTimer) clearTimeout(releaseScrollTargetTimer)
  releaseScrollTargetTimer = setTimeout(() => {
    scrollIntoView.value = ''
  }, 160)
}
```

**Step 4: Wire explicit stick-to-bottom events**

In `onSend()`:

```ts
isThinking.value = true
const sending = chatStore.sendMessage()
requestStickToBottom()
await sending
isThinking.value = false
requestStickToBottom()
```

In `onRetry()`:

```ts
const sending = chatStore.sendMessage()
requestStickToBottom()
await sending
requestStickToBottom()
```

In `onNewChat()` and `onSelectSession()` after the new messages are available:

```ts
requestStickToBottom(false)
```

**Step 5: Add passive auto-stick watcher**

Add:

```ts
watch(
  () => [
    chatStore.messages[0]?.id ?? 'empty',
    chatStore.messages.map((m: Message) => m.id).join('|'),
    isThinking.value ? 'thinking' : 'idle',
  ],
  ([firstMessageId], previous) => {
    const conversationChanged = firstMessageId !== previous?.[0]
    if (conversationChanged) {
      autoStickToBottom.value = true
      requestStickToBottom(false)
      return
    }
    void scrollToBottom(false, true)
  },
  { immediate: true },
)
```

This preserves the prototype behavior:

- Initial load and session switches go to the bottom.
- User-initiated sends go to the bottom.
- Ordinary message changes only auto-stick if the user is already near bottom.
- If the user has swiped upward, passive updates do not steal scroll.

**Step 6: Clean up timer on unmount**

In `onUnmounted()`:

```ts
if (releaseScrollTargetTimer) clearTimeout(releaseScrollTargetTimer)
```

**Step 7: Run layout test**

Run:

```bash
cd frontend
npm test -- src/pages/chat/index-layout.test.ts
```

Expected: pass.

---

## Task 4: Apply Miniapp Layout Styles

**Files:**
- Modify: `frontend/src/pages/chat/index.scss`
- Test: `frontend/src/pages/chat/index-layout.test.ts`

**Step 1: Add explicit layout variables**

Use:

```scss
.chat-page {
  --chat-tabbar-reserve: 220px;
  --chat-composer-height: 112px;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  height: 100vh;
  height: 100dvh;
  padding-bottom: 0;
  background: #1c1b18;
  overflow: hidden;
}
```

`--chat-tabbar-reserve` protects the composer from the custom tabbar.

`--chat-composer-height` protects the last message from the overlaid composer.

**Step 2: Make chat body the positioning scope**

Use:

```scss
.chat-body {
  position: relative;
  flex: 1 1 0;
  min-height: 0;
  box-sizing: border-box;
  overflow: hidden;
  padding-bottom: calc(var(--chat-tabbar-reserve) + env(safe-area-inset-bottom));
}
```

**Step 3: Make scroll-view fill the body**

Use:

```scss
.chat-scroll-area {
  height: 100%;
  min-height: 0;
  width: 100%;
  box-sizing: border-box;
  overflow-y: auto;
  overflow-x: hidden;
  -webkit-overflow-scrolling: touch;
}

.chat-scroll-area .messages-inner {
  box-sizing: border-box;
  min-height: 100%;
  padding: 24px 20px 0;
}
```

**Step 4: Add composer spacer**

Use:

```scss
.composer-spacer {
  height: var(--chat-composer-height);
  flex-shrink: 0;
}

.chat-scroll-bottom-anchor {
  height: 1px;
}
```

**Step 5: Overlay composer above tabbar**

Use:

```scss
.chat-composer-slot {
  position: absolute;
  left: 0;
  right: 0;
  bottom: calc(var(--chat-tabbar-reserve) + env(safe-area-inset-bottom));
  width: 100%;
  box-sizing: border-box;
  z-index: 20;
}
```

Keep the existing `ChatInput.vue` `.composer` styles unless the measured composer height does not match `--chat-composer-height`.

**Step 6: Verify no old scroll shell remains**

Search:

```bash
rg "chat-message-panel|stickToBottomRequest|viewportHeight|scroll-top" frontend/src/pages/chat frontend/src/components/chat
```

Expected:

- No `chat-message-panel`.
- No `stickToBottomRequest`.
- No `viewportHeight` prop on `MessageList`.
- No `scroll-top` control for the chat.

---

## Task 5: Verification

**Files:**
- Verify only unless fixes are required.

**Step 1: Run focused tests**

Run:

```bash
cd frontend
npm test -- src/pages/chat/index-layout.test.ts src/components/chat/MessageList.test.ts src/components/chat/ChatInput.test.ts
```

Expected:

- All tests pass.

**Step 2: Run typecheck**

Run:

```bash
cd frontend
npm run typecheck
```

Expected:

- Exit code 0.

**Step 3: Refresh miniapp build output**

If a `dev:weapp --watch` process is already running, verify `frontend/dist/pages/chat/index.js` and `frontend/dist/pages/chat/index.wxss` update automatically.

Otherwise run:

```bash
cd frontend
npm run build:weapp
```

Known caveat: this repository has previously shown a macOS `system-configuration` Rust panic during Taro build. If it appears, do not claim build success. Use source tests, typecheck, and dist key-line inspection as partial verification.

**Step 4: Inspect dist key lines**

Run:

```bash
rg -n "chat-scroll-area|composer-spacer|chat-scroll-bottom|chat-message-panel|stickToBottomRequest|viewportHeight|scroll-top" frontend/dist/pages/chat/index.js frontend/dist/pages/chat/index.wxss
```

Expected:

- Present: `chat-scroll-area`
- Present: `composer-spacer`
- Present: `chat-scroll-bottom`
- Absent: `chat-message-panel`
- Absent: `stickToBottomRequest`
- Absent: `scroll-top`

---

## Manual WeChat Miniapp QA Checklist

Use WeChat DevTools or the real miniapp preview.

1. Open the chat tab.
2. Confirm the input composer is fully visible above the custom tabbar.
3. Ask at least three questions so the conversation exceeds one screen.
4. Confirm the latest user bubble and assistant answer stop above the input composer, not behind it.
5. Swipe up from the message area.
6. Confirm older messages can be revealed, including the initial welcome message.
7. While scrolled upward, wait for any passive state update. Confirm the view does not jump to bottom unless the user sends a new question.
8. Send a new question. Confirm the chat returns to the bottom.
9. Tap “新聊天”. Confirm it opens at the bottom of the new conversation.
10. Open history and switch sessions. Confirm restored session starts at the bottom and can be swiped upward.
11. Test an iPhone safe-area device.
12. Test an Android or no-safe-area device.

---

## If The Composer Height Changes Later

This plan assumes a single-line composer with a stable height. If `ChatInput` later supports multi-line input, voice input, attachments, or keyboard-aware resizing, replace the fixed CSS variable with measured composer height:

- Measure `.chat-composer-slot`.
- Store height in `composerHeight`.
- Apply it to scroll content spacer via inline style or CSS variable.
- Re-run the same manual QA.

Do not remove the spacer. The spacer is the miniapp adaptation that prevents the last message from being covered by the overlaid composer.
