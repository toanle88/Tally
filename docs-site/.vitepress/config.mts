import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/Tally/',
  title: 'TALLY',
  description: 'A finance platform built as a solo, part-time learning project.',
  ignoreDeadLinks: false,
  themeConfig: {
    nav: [
      { text: 'Overview', link: '/' },
      { text: 'Current status', link: '/current-status' },
      { text: 'Architecture', link: '/architecture' },
      { text: 'Security & IAM', link: '/identity-access' },
      { text: 'Operations', link: '/operations' },
      { text: 'Roadmap', link: '/roadmap' },
      { text: 'GitHub', link: 'https://github.com/toanle88/Tally' },
    ],
    sidebar: [
      { text: 'Project overview', link: '/' },
      { text: 'Current status & evidence', link: '/current-status' },
      { text: 'Architecture', link: '/architecture' },
      { text: 'Finance integrity', link: '/finance-integrity' },
      { text: 'Identity & access', link: '/identity-access' },
      { text: 'Operations & delivery', link: '/operations' },
      { text: 'UX foundation', link: '/ux' },
      { text: 'Roadmap & progress', link: '/roadmap' },
      { text: 'Local development', link: '/development' },
      { text: 'Learning notes & decisions', link: '/learning-notes' },
      { text: 'Authoritative sources', link: '/sources' },
    ],
    search: { provider: 'local' },
    editLink: { pattern: 'https://github.com/toanle88/Tally/edit/main/docs-site/:path' },
    socialLinks: [{ icon: 'github', link: 'https://github.com/toanle88/Tally' }],
  },
})
