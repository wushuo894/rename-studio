import {ChevronDown, ChevronUp} from '@lucide/vue'
import {createApp} from 'vue'
import {createVuetify} from 'vuetify'
import {VApp, VBtn, VDialog, VMenu, VNumberInput, VSelect, VSnackbar, VTextField, VTooltip,} from 'vuetify/components'
import {Ripple} from 'vuetify/directives'
import 'vuetify/styles'
import App from './App.vue'
import './styles.css'

const vuetify = createVuetify({
    // 显式注册组件和指令，避免运行时模板中的 v-text-field 等控件被当作未知标签。
    components: {
        VApp, VBtn, VDialog, VMenu, VNumberInput, VSelect,
        VSnackbar, VTextField, VTooltip,
    },
    directives: {Ripple},
    // 数字步进按钮使用项目内置的 Lucide 图标，无需加载图标字体。
    icons: {aliases: {collapse: ChevronUp, expand: ChevronDown}},
    theme: {
        defaultTheme: 'renameDark',
        themes: {
            renameDark: {
                dark: true,
                colors: {
                    background: '#11151b',
                    surface: '#1a2029',
                    'surface-variant': '#252e3a',
                    primary: '#85b7ff',
                    secondary: '#a7b0be',
                    error: '#ff6b78',
                    warning: '#f4bd62',
                    success: '#54c59a',
                },
            },
        },
    },
    defaults: {
        VBtn: {rounded: 'lg', elevation: 0},
        VNumberInput: {density: 'comfortable', controlVariant: 'stacked', hideDetails: 'auto', color: 'primary'},
        VTextField: {density: 'comfortable', variant: 'outlined', hideDetails: 'auto', color: 'primary'},
        VSelect: {density: 'comfortable', variant: 'outlined', hideDetails: 'auto', color: 'primary'},
    },
})

createApp(App).use(vuetify).mount('#app')
