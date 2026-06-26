# Chat Page WeChat Layout Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the digital clerk chat page behave like a WeChat conversation, with an independent message scroll area and a fixed composer that never overlaps messages.

**Architecture:** Keep the page as a vertical flex shell. The message list remains the only scrollable region, while the composer is fixed above the existing custom tab bar. Reserve exactly the composer and tab bar heights in the conversation area so short conversations start at the top and long conversations auto-scroll to the newest message at the bottom.

**Tech Stack:** Taro + Vue 3 + SCSS + Vitest string-based layout regression tests.

---

### Task 1: Update Layout Regression Expectations

**Files:**
- Modify: `frontend/src/pages/chat/index-layout.test.ts`
- Modify: `frontend/src/components/chat/MessageList.test.ts`

**Step 1: Write the failing test**

Assert that the chat layout uses explicit page-level CSS variables for tab bar and composer height, reserves both for the fixed composer, places the composer at `104px + safe-area`, and does not retain the stale `220px` chat offset.

**Step 2: Run test to verify it fails**

Run: `npm test -- index-layout.test.ts MessageList.test.ts`

Expected: FAIL because the current chat stylesheet still uses `220px` offsets.

### Task 2: Implement Minimal Layout Fix

**Files:**
- Modify: `frontend/src/pages/chat/index.scss`
- Modify only if needed: `frontend/src/components/chat/MessageList.vue`

**Step 1: Update chat layout SCSS**

Set chat layout variables:

```scss
--chat-tabbar-height: 104px;
--chat-composer-reserved-height: 112px;
```

Use them for:

```scss
padding-bottom: calc(var(--chat-composer-reserved-height) + var(--chat-tabbar-height) + env(safe-area-inset-bottom));
bottom: calc(var(--chat-tabbar-height) + env(safe-area-inset-bottom));
```

**Step 2: Keep message list top-flowing and bottom-scrollable**

Keep `.messages-inner { min-height: 100%; }` and the existing `scroll-bottom` anchor so short chats start from the top and long chats scroll to the latest message.

**Step 3: Run focused tests**

Run: `npm test -- index-layout.test.ts MessageList.test.ts`

Expected: PASS.

### Task 3: Verify Frontend Safety

**Files:**
- All touched frontend files.

**Step 1: Run all frontend tests**

Run: `npm test`

Expected: PASS.

**Step 2: Typecheck if practical**

Run: `npm run typecheck`

Expected: PASS, or report any pre-existing unrelated failures.
