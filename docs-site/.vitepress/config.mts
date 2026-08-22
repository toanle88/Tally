import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/Tally/',
  title: 'TALLY',
  description: 'A finance platform built as a solo, part-time learning project.',
  ignoreDeadLinks: false,
  themeConfig: {
    nav: [
      { text: 'Overview', link: '/' },
      { text: 'Architecture', link: '/architecture' },
      { text: 'Roadmap', link: '/roadmap' },
      { text: 'GitHub', link: 'https://github.com/toanle88/Tally' },
    ],
    sidebar: [
      { text: 'Project overview', link: '/' },
      { text: 'Architecture', link: '/architecture' },
      { text: 'Roadmap & progress', link: '/roadmap' },
      { text: 'Local development', link: '/development' },
      { text: 'Finance integrity', link: '/finance-integrity' },
      { text: 'Learning notes & decisions', link: '/learning-notes' },
      { text: 'Authoritative sources', link: '/sources' },
    ],
    search: { provider: 'local' },
    editLink: { pattern: 'https://github.com/toanle88/Tally/edit/main/docs-site/:path' },
    socialLinks: [{ icon: 'github', link: 'https://github.com/toanle88/Tally' }],
  },
})
