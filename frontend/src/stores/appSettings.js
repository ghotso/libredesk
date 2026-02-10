import { defineStore } from 'pinia'
import api from '@/api'

export const useAppSettingsStore = defineStore('settings', {
    state: () => ({
        settings: {},
        public_config: {}
    }),
    actions: {
        async fetchSettings (key = 'general') {
            try {
                const response = await api.getSettings(key)
                // Support both envelope { data: payload } and direct payload
                const data = response?.data?.data ?? response?.data ?? {}
                this.settings = data
                return this.settings
            } catch (error) {
                // Pass
            }
        },
        async fetchPublicConfig () {
            try {
                const response = await api.getConfig()
                this.public_config = response?.data?.data || {}
                return this.public_config
            } catch (error) {
                // Pass
            }
        },
        setSettings (newSettings) {
            this.settings = newSettings
        },
        setPublicConfig (newPublicConfig) {
            this.public_config = newPublicConfig
        }
    }
})
