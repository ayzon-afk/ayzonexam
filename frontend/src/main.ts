import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'
import { applyTheme, loadTheme } from './lib/theme'

// Тема — до первого кадра, чтобы экран не мигнул чужими цветами.
applyTheme(loadTheme())

mount(App, { target: document.getElementById('app')! })
