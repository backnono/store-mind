import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './app.scss'

const App = createApp({
  onShow() {
    // App shown
  },
})

const pinia = createPinia()
App.use(pinia)

export default App
