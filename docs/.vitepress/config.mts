import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "nanogit",
  description: "Lightweight, HTTPS-only Git implementation for cloud-native environments",
  base: '/nanogit/',

  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    logo: '/logo.svg',

    nav: [
      { text: 'Home', link: '/' },
      { text: 'API Reference', link: 'https://pkg.go.dev/github.com/grafana/nanogit' }
    ],

    sidebar: [
      {
        text: 'Getting Started',
        items: [
          { text: 'Installation', link: '/getting-started/installation' },
          { text: 'Quick Start', link: '/getting-started/quick-start' }
        ]
      },
      {
        text: 'Architecture',
        items: [
          { text: 'Overview', link: '/architecture/overview' },
          { text: 'Storage Backend', link: '/architecture/storage' },
          { text: 'Delta Resolution', link: '/architecture/delta-resolution' },
          { text: 'Performance', link: '/architecture/performance' }
        ]
      },
      {
        text: 'More',
        items: [
          { text: 'Changelog', link: '/changelog' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/grafana/nanogit' }
    ],

    editLink: {
      pattern: 'https://github.com/grafana/nanogit/edit/main/docs/:path',
      text: 'Edit this page on GitHub'
    },

    footer: {
      message: 'Released under the Apache 2.0 License.',
      copyright: 'Copyright © Grafana Labs'
    },

    search: {
      provider: 'local'
    },

    outline: {
      level: [2, 3]
    }
  },

  markdown: {
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    },
    lineNumbers: true
  }
})
