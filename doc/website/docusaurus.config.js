/** @type {import('@docusaurus/types').DocusaurusConfig} */
module.exports = {
  title: 'Atlas',
  tagline: 'Manage your data',
  url: 'https://atlasgo.io',
  baseUrl: '/',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  organizationName: 'ariga',
  projectName: 'atlas',
  themeConfig: {
    prism: {
      additionalLanguages: ['hcl'],
    },
    algolia: {
      appId: 'D158RRDJO1',
      apiKey: "383b9efbb1486f812c8c023556b15396",
      indexName: "atlasgo",
    },
    navbar: {
      title: 'Atlas',
      logo: {
        alt: 'Atlas',
        src: 'https://atlasgo.io/uploads/landing/logo.svg',
      },
      items: [
        {
          href: 'https://github.com/ariga/atlas',
          className: 'header-github-link',
          position: 'right',
        },
        {
          href: 'https://discord.gg/zZ6sWVg6NT',
          className: 'header-discord-link',
          position: 'right',
        },{
          href: 'https://twitter.com/ariga_io',
          className: 'header-twitter-link',
          position: 'right',
        },
        {
          to: 'cli/getting-started/setting-up',
          label: 'Docs',
          position: 'left',
        },
        {
          to: 'knowledge',
          label: 'Knowledge Center',
          position: 'left',
        },
        {
          to: 'blog',
          label: 'Blog',
          position: 'left'
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
            {"label": "CLI Reference", "to": "cli/reference"},
            {"label": "Blog", "to": "blog"},
            {"label": "Knowledge Center", "to": "knowledge"},
            {"label": "About", "to": "about"},
          ]
        },
        {
          "title": "Community",
          "items": [
            {"label": "GitHub", "to": "https://github.com/ariga/atlas"},
            {"label": "Discord", "to": "https://discord.gg/zZ6sWVg6NT"},
          ]
        },
        {
          "title": "Legal",
          "items": [
            {"label": "Privacy Policy", "to": "https://ariga.io/legal/privacy"},
            {"label": "Terms of Service", "to": "https://ariga.io/legal/tos"},
            {"label": "End User License", "to": "https://ariga.io/legal/atlas/eula"},
          ]
        }
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
    announcementBar: {
      id: 'announcementBar-1', // Increment on change
      content: `️🚀 Sign up for a user testing session and receive exclusive Atlas swag, register <a target="_blank" rel="noopener noreferrer" href="https://calendly.com/ariga-user-testing/atlas-user-testing">here!</a>`,
      isCloseable: true,
    },
  },
  plugins: [
    [
      '@docusaurus/plugin-client-redirects',
      {
        redirects: [
          {
            to: '/dev-database',
            from: '/cli/dev-database',
          },
        ],
      },
    ],
    [
      require.resolve('docusaurus-gtm-plugin'),
      {
        id: 'GTM-T9GX8BR', // GTM Container ID
      }
    ],
  ],
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
        },
        gtag: {
          trackingID: 'G-Z88N4TF03R'
        },
        blog: {
          showReadingTime: true,
          blogSidebarTitle: 'All our posts',
          blogSidebarCount: 'ALL',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      },
    ],
  ],
};
