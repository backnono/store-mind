import path from 'node:path'
import { defineConfig } from '@tarojs/cli'

const config = defineConfig({
  projectName: 'store-mind-miniapp',
  date: '2026-06-22',
  sourceRoot: 'src',
  outputRoot: 'dist',
  framework: 'vue3',
  alias: {
    '@': path.resolve(__dirname, '..', 'src'),
  },
  compiler: {
    type: 'webpack5',
    prebundle: { enable: false },
  },
  sass: {
    data: '$primary: oklch(58% 0.16 70); $primary-dark: oklch(48% 0.18 85);',
  },
})

export default config
