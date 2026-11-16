import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'
import { initTheme } from './lib/theme'

// Тема — до первого кадра, чтобы экран не мигнул чужими цветами.
initTheme()

mount(App, { target: document.getElementById('app')! })
