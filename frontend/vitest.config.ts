import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import path from 'path'

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/**/*.test.ts'],
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      '@tarojs/taro': path.resolve(__dirname, 'src/__mocks__/taro.ts'),
    },
  },
  define: {
    ENABLE_INNER_HTML: 'false',
    ENABLE_SIZE_APIS: 'false',
    ENABLE_TEMPLATE_CONTENT: 'false',
    ENABLE_CLONE_NODE: 'false',
    ENABLE_CONTAINS: 'false',
    ENABLE_MUTATION_OBSERVER: 'false',
    ENABLE_INPUT: 'false',
    ENABLE_TOUCH: 'false',
    ENABLE_CANVAS: 'false',
    ENABLE_CSS_TRANSITION: 'false',
    ENABLE_SCROLL: 'false',
  },
})
