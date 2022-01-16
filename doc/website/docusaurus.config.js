/** @type {import('@docusaurus/types').DocusaurusConfig} */
module.exports = {
  title: 'Atlas',
  tagline: 'Manage your data',
  url: 'https://atlasgo.io',
  baseUrl: '/',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  favicon: 'https://atlasgo.io/uploads/favicon.icon',
  organizationName: 'ariga', // Usually your GitHub org/user name.
  projectName: 'atlas', // Usually your repo name.
  themeConfig: {
    gtag: {
      trackingID: 'G-Z88N4TF03R'
    },
    prism: {
      additionalLanguages: ['hcl'],
    },
    navbar: {
      title: 'Atlas',
      logo: {
        alt: 'Atlas',
        src: 'https://atlasgo.io/uploads/landing/logo.svg',
      },
      items: [
        // {
        //   type: 'doc',
        //   docId: 'getting-started',
        //   position: 'left',
        //   label: 'Welcome',
        // },
        // {to: '/blog', label: 'Docs', position: 'left'},
        // {to: '/blog', label: 'Tutorials', position: 'left'},
        // {to: '/blog', label: 'Atlas For Ent', position: 'left'},
        // {to: '/blog', label: 'Getting Started', position: 'left'},
        {
          href: 'https://github.com/ariga/atlas',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    //
    "footer": {
      "links": [
        {
          "title": "Docs",
          "items": [
            {"label": "Getting Started", "to": "cli/getting-started/setting-up"},
            {"label": "Data Definition Language ", "to": "ddl/intro"},
            {"label": "CLI Reference", "to": "CLI/atlas"},
          ]
        },
        {
          "title": "Community",
          "items": [
            {"label": "GitHub", "to": "https://github.com/ariga/atlas"},
            {"label": "Discord", "to": "https://discord.com/QhsmBAWzrC"},
          ]
        },
      ],
      copyright: `
      Copyright © ${new Date().getFullYear()} The Atlas Authors.
      The Go gopher was designed by <a href="http://reneefrench.blogspot.com/">Renee French</a>.
      <br/>
      The design is licensed under the Creative Commons 3.0 Attributions license. Read this 
      <a href="https://blog.golang.org/gopher">article</a> for more details.
      <br/>
      `,
    },
  },
  presets: [
    [
      '@docusaurus/preset-classic',
      {
        docs: {
          routeBasePath: '/',
          sidebarPath: require.resolve('./sidebars.js'),
          path: "../md",
          showLastUpdateAuthor: false,
          showLastUpdateTime: false,
          // editUrl:
          //   'https://github.com/ariga/atlas/edit/master/doc/website/',
        },
        blog: {
          showReadingTime: true,
          // Please change this to your repo.
          // editUrl:
          //   'https://github.com/facebook/docusaurus/edit/master/website/blog/',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      },
    ],
  ],
};
