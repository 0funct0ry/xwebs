import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'xwebs',
  tagline: 'WebSocket Swiss Army Knife',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'https://0funct0ry.github.io',
  baseUrl: '/xwebs/',

  organizationName: '0funct0ry',
  projectName: 'xwebs',

  onBrokenLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/0funct0ry/xwebs/tree/main/docs-site/',
          showLastUpdateTime: true,
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/docusaurus-social-card.jpg',
    colorMode: {
      defaultMode: 'light',
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'xwebs',
      logo: {
        alt: 'xwebs Logo',
        src: 'img/logo.svg',
      },
      style: 'primary',
      hideOnScroll: false,
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docs',
          position: 'left',
          label: 'Docs',
        },
        {
          to: '/docs/examples/',
          label: 'Examples',
          position: 'left',
        },
        {
          to: '/docs/reference/cli',
          label: 'CLI Reference',
          position: 'left',
        },
        {
          href: 'https://github.com/0funct0ry/xwebs',
          position: 'right',
          className: 'header-github-link',
          'aria-label': 'GitHub repository',
          label: 'GitHub',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Learn',
          items: [
            {label: 'Getting Started',      to: '/docs/getting-started'},
            {label: 'Handler YAML Schema',  to: '/docs/handlers/yaml-schema'},
            {label: 'Template Functions',   to: '/docs/templates/functions'},
            {label: 'Builtins',            to: '/docs/builtins/'},
          ],
        },
        {
          title: 'Reference',
          items: [
            {label: 'CLI Reference',      to: '/docs/reference/cli'},
            {label: 'Binary Frames',      to: '/docs/reference/binary-frames'},
            {label: 'Examples',           to: '/docs/examples/'},
          ],
        },
        {
          title: 'Community',
          items: [
            {label: 'GitHub',  href: 'https://github.com/0funct0ry/xwebs'},
            {label: 'Issues',  href: 'https://github.com/0funct0ry/xwebs/issues'},
            {label: 'Releases',href: 'https://github.com/0funct0ry/xwebs/releases'},
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} xwebs contributors — MIT License`,
    },
    prism: {
      theme: prismThemes.oneLight,
      darkTheme: prismThemes.oneDark,
      additionalLanguages: ['bash', 'yaml', 'go', 'lua', 'json', 'protobuf'],
      defaultLanguage: 'bash',
    },
    docs: {
      sidebar: {
        hideable: true,
        autoCollapseCategories: true,
      },
    },
    algolia: undefined,
  } satisfies Preset.ThemeConfig,
};

export default config;
