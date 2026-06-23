export default defineAppConfig({
  pages: [
    'pages/store/index',
    'pages/shop/index',
    'pages/chat/index',
    'pages/orders/index',
    'pages/profile/index',
  ],
  window: {
    navigationStyle: 'custom',
    backgroundColor: '#1c1b18',
  },
  tabBar: {
    custom: true,
    color: '#7a6e5c',
    selectedColor: '#d4a44c',
    backgroundColor: '#1c1b18',
    borderStyle: 'black',
    list: [
      { pagePath: 'pages/store/index', text: '首页' },
      { pagePath: 'pages/shop/index', text: '购物' },
      { pagePath: 'pages/chat/index', text: '小王' },
      { pagePath: 'pages/orders/index', text: '订单' },
      { pagePath: 'pages/profile/index', text: '我的' },
    ],
  },
})
