import {createApp} from 'vue'
import {createVuetify} from 'vuetify'
import {app as neutralinoApp, events, init} from '@neutralinojs/lib'
import {VApp, VBtn, VSelect, VSnackbar, VSpacer, VTextField, VTooltip,} from 'vuetify/components'
import {Ripple} from 'vuetify/directives'
import 'vuetify/styles'
import App from './App.vue'
import './styles.css'

const vuetify = createVuetify({
    // 显式注册组件和指令，避免运行时模板中的 v-text-field 等控件被当作未知标签。
    components: {
        VApp, VBtn, VSelect,
        VSnackbar, VSpacer, VTextField, VTooltip,
    },
    directives: {Ripple},
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
        VBtn: {rounded: 'sm'},
        VTextField: {density: 'comfortable', variant: 'outlined', hideDetails: 'auto', color: 'primary'},
        VSelect: {density: 'comfortable', variant: 'outlined', hideDetails: 'auto', color: 'primary'},
    },
})

/** 配置桌面窗口的正常退出流程，绕开 macOS 直接关闭原生窗口时的主线程死锁。 */
function configureDesktopLifecycle() {
    let exiting = false

    /** 所有退出入口复用 app.exit，保证 Neutralino 服务和 WebView 按正常流程停止。 */
    const exitApplication = () => {
        if (exiting) return
        exiting = true
        neutralinoApp.exit()
    }

    init()
    events.on('windowClose', exitApplication)

    // 键盘监听作为非原生菜单环境的后备路径，浏览器预览不会注册此监听器。
    window.addEventListener('keydown', (event) => {
        if (!event.metaKey || !['q', 'w'].includes(event.key.toLocaleLowerCase())) return
        event.preventDefault()
        exitApplication()
    })
}

// 浏览器预览不包含 Neutralino 全局变量，仅在桌面容器中建立原生 API 连接。
if (typeof window.NL_PORT !== 'undefined') configureDesktopLifecycle()

createApp(App).use(vuetify).mount('#app')
