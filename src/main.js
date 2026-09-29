import { createApp } from 'vue'
import { createVuetify } from 'vuetify'
import {
  VApp, VBtn, VSelect,
  VSnackbar, VSpacer, VTextField, VTooltip,
} from 'vuetify/components'
import { Ripple } from 'vuetify/directives'
import 'vuetify/styles'
import App from './App.vue'
import './styles.css'

const vuetify = createVuetify({
  // 显式注册组件和指令，避免运行时模板中的 v-text-field 等控件被当作未知标签。
  components: {
    VApp, VBtn, VSelect,
    VSnackbar, VSpacer, VTextField, VTooltip,
  },
  directives: { Ripple },
  theme: {
    defaultTheme: 'renameDark',
    themes: {
      renameDark: {
        dark: true,
        colors: {
          background: '#171a1f',
          surface: '#20242a',
          'surface-variant': '#292e35',
          primary: '#58a6ff',
          secondary: '#a7b0be',
          error: '#ff6b78',
          warning: '#f4bd62',
          success: '#54c59a',
        },
      },
    },
  },
  defaults: {
    VBtn: { rounded: 'sm' },
    VTextField: { density: 'comfortable', variant: 'outlined', hideDetails: 'auto', color: 'primary' },
    VSelect: { density: 'comfortable', variant: 'outlined', hideDetails: 'auto', color: 'primary' },
  },
})

createApp(App).use(vuetify).mount('#app')
