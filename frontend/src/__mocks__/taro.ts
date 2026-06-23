// Mock for @tarojs/taro — provides enough API surface
// for unit tests that don't need a real mini program runtime.

const noop = () => undefined
const asyncNoop = () => Promise.resolve(undefined)

const storage: Record<string, unknown> = {}

export default {
  request: () => Promise.resolve({ statusCode: 200, data: {} }),
  navigateTo: noop,
  navigateBack: noop,
  redirectTo: noop,
  switchTab: noop,
  reLaunch: noop,
  getCurrentPages: () => [],
  setStorage: ({ key, data }: { key: string; data: unknown }) => {
    storage[key] = data
    return Promise.resolve()
  },
  getStorage: ({ key }: { key: string }) => {
    if (key in storage) return Promise.resolve({ data: storage[key] })
    return Promise.reject(new Error('not found'))
  },
  removeStorage: ({ key }: { key: string }) => {
    delete storage[key]
    return Promise.resolve()
  },
  clearStorage: () => {
    Object.keys(storage).forEach((k) => delete storage[k])
    return Promise.resolve()
  },
  getStorageInfo: () =>
    Promise.resolve({
      keys: Object.keys(storage),
      currentSize: 0,
      limitSize: 10240,
    }),
  scanCode: () => Promise.reject(new Error('scanCode:fail cancel')),
  showToast: noop,
  hideToast: noop,
  showLoading: noop,
  hideLoading: noop,
  showModal: () => Promise.resolve({ confirm: true, cancel: false }),
  getSystemInfoSync: () => ({
    platform: 'devtools',
    model: 'iPhone 14',
    pixelRatio: 3,
    windowWidth: 390,
    windowHeight: 844,
    system: 'iOS 17',
  }),
  getSystemInfo: () =>
    Promise.resolve({
      platform: 'devtools',
      model: 'iPhone 14',
    }),
  eventCenter: {
    on: noop,
    off: noop,
    trigger: noop,
  },
  nextTick: (fn?: () => void) => {
    if (fn) setTimeout(fn, 0)
    return Promise.resolve()
  },
  useLoad: (fn: (...args: unknown[]) => void) => fn,
  useDidShow: asyncNoop,
  useDidHide: asyncNoop,
  useUnload: asyncNoop,
}

export function useLoad(fn: (...args: unknown[]) => void) {
  return fn
}
