import { defineStore } from 'pinia'
import request from '../utils/request'

export interface SiteTheme {
  primary_color: string
  bg_color: string
  card_radius: string
  list_layout: string
  theme_mode: string
  custom_css: string
}

export const useThemeStore = defineStore('theme', {
  state: () => ({
    config: null as SiteTheme | null,
    loaded: false,
  }),

  actions: {
    async fetchTheme() {
      try {
        const res = await request.get<SiteTheme>('/theme')
        this.config = res
        this.loaded = true
        this.applyTheme()
      } catch (e) {
        console.error('获取主题配置失败', e)
        this.loaded = true
      }
    },

    applyTheme() {
      if (!this.config) return
      const root = document.documentElement

      // 注入 CSS 变量
      root.style.setProperty('--c-primary', this.config.primary_color)
      root.style.setProperty('--c-bg', this.config.bg_color)
      root.style.setProperty('--radius-base', this.config.card_radius)

      // 暗色/亮色模式
      if (this.config.theme_mode === 'dark') {
        root.classList.add('dark')
      } else {
        root.classList.remove('dark')
      }

      // 注入自定义 CSS
      if (this.config.custom_css) {
        let styleEl = document.getElementById('custom-theme-css')
        if (!styleEl) {
          styleEl = document.createElement('style')
          styleEl.id = 'custom-theme-css'
          document.head.appendChild(styleEl)
        }
        styleEl.innerHTML = this.config.custom_css
      } else {
        const styleEl = document.getElementById('custom-theme-css')
        if (styleEl) {
          styleEl.remove()
        }
      }
    },

    // 预览主题（不持久化，仅用于后台管理预览）
    previewTheme(theme: SiteTheme) {
      const root = document.documentElement
      root.style.setProperty('--c-primary', theme.primary_color)
      root.style.setProperty('--c-bg', theme.bg_color)
      root.style.setProperty('--radius-base', theme.card_radius)
      if (theme.theme_mode === 'dark') {
        root.classList.add('dark')
      } else {
        root.classList.remove('dark')
      }
    },

    // 重置预览（恢复已保存的主题）
    resetPreview() {
      this.applyTheme()
    },
  },
})
