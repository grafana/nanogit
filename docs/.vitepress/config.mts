import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "nanogit",
  description: "Lightweight, HTTPS-only Git implementation for cloud-native environments",
  base: '/nanogit/',

  // Allow links to testutil package outside docs directory
  ignoreDeadLinks: [
    /testutil\/examples\/.*/,
    /testutil\/README\.md/
  ],

  head: [
    ['link', { rel: 'icon', type: 'image/png', href: '/nanogit/logo.png' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'nanogit' }],
    ['meta', { property: 'og:description', content: 'Lightweight, HTTPS-only Git implementation for cloud-native environments' }],
    ['meta', { property: 'og:image', content: 'https://grafana.github.io/nanogit/logo.png' }],
  ],

  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    logo: '/logo.png',

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
