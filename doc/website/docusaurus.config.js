/** @type {import('@docusaurus/types').DocusaurusConfig} */
module.exports = {
  title: 'Atlas | Open-source database schema management tool',
  tagline: 'Manage your database schemas with Atlas CLI',
  url: 'https://atlasgo.io',
  baseUrl: '/',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  organizationName: 'ariga',
  projectName: 'atlas',
  themeConfig: {
    image: "https://atlas-og-img.vercel.app/**Atlas%20%7C**%20Open-source%20database%20schema%20management.png?theme=dark",
    prism: {
      additionalLanguages: ['hcl'],
      magicComments: [
        {
          className: 'theme-code-block-highlighted-line',
          line: 'highlight-next-line',
          block: {start: 'highlight-start', end: 'highlight-end'},
        },
        {
          className: 'code-block-error-message',
          line: 'highlight-next-line-error-message',
        },
        {
          className: 'code-block-info-line',
          line: 'highlight-next-line-info',
        },
      ],
    },
    algolia: {
      appId: 'D158RRDJO1',
      apiKey: "3585a7e658ef4ab7775a9099be3778d2",
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
        },{
          href: 'https://atlasnewsletter.substack.com/',
          className: 'header-newsletter-link',
          position: 'right',
        },
        
        {
          to: 'getting-started',
          label: 'Docs',
          position: 'left',
        },
        {
          to: 'guides',
          label: 'Guides',
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
            {"label": "Getting Started", "to": "getting-started"},
            {"label": "Data Definition Language ", "to": "guides/ddl"},
            {"label": "CLI Reference", "to": "cli-reference"},
            {"label": "Blog", "to": "blog"},
            {"label": "Guides", "to": "guides"},
            {"label": "About", "to": "about"},
            {"label": "GoDoc", "to": "https://pkg.go.dev/ariga.io/atlas"},
          ]
        },
        {
          "title": "Community",
          "items": [
            {"label": "GitHub", "to": "https://github.com/ariga/atlas"},
            {"label": "Discord", "to": "https://discord.gg/zZ6sWVg6NT"},
            {"label": "Twitter", "to": "https://twitter.com/ariga_io"},
            {"label": "Newsletter", "to": "https://atlasnewsletter.substack.com/"},
            {"label": "YouTube", "to": "https://youtube.com/@ariga_io"}
          ]
        },
        {
          "title": "Integrations",
          "items": [
            {"label": "GitHub Actions", "to": "/integrations/github-actions"},
            {"label": "Terraform", "to": "/integrations/terraform-provider"},
            {"label": "Go API", "to": "/integrations/go-api"}
          ]
        },
        {
          "title": "Legal",
          "items": [
            {"label": "Privacy Policy", "to": "https://ariga.io/legal/privacy"},
            {"label": "Terms of Service", "to": "https://ariga.io/legal/tos"},
          ]
        }
      ],
      copyright: `
      Copyright © ${new Date().getFullYear()} The Atlas Authors.
      The Go gopher was designed by <a href="http://reneefrench.blogspot.com/">Renee French</a>.
      <br/>
      The design for the Go gopher is licensed under the Creative Commons 3.0 Attributions license. Read this 
      <a href="https://blog.golang.org/gopher">article</a> for more details.
      <br/>
      `,
    },
    announcementBar: {
      id: 'announcementBar-3', // Increment on change
      content: `Atlas v0.10 is now available! Check out the <a href="/blog/2023/03/13/atlas-v010">latest blog post</a> 🚀`,
      isCloseable: true,
    },
  },
  plugins: [
    [
      '@docusaurus/plugin-client-redirects',
      {
        redirects: [
          {
            to: '/getting-started/',
            from: '/cli/getting-started/setting-up',
          },
          {
            to: '/integrations/terraform-provider',
            from: '/terraform-provider'
          },
          {
            to: '/integrations/go-api',
            from: ['/go-api/intro','/go-api/inspect'],
          },
          {
            to: '/cli-reference',
            from: '/cli/reference',
          },
          {
            to: '/concepts/url',
            from: '/cli/url',
          },
          {
            to: '/concepts/dev-database',
            from: '/dev-database',
          },
          {
            to: '/guides/ddl',
            from: ['/ddl/intro', '/concepts/ddl'],
          },
          {
            to: '/atlas-schema/input-variables',
            from: '/ddl/input-variables',
          },
          {
            to: '/atlas-schema/projects',
            from: '/cli/projects',
          },
          {
            to: '/atlas-schema/sql-types',
            from: '/ddl/sql-types',
          },
          {
            to: '/atlas-schema/sql-resources',
            from: '/ddl/sql',
          },
          {
            to: '/guides',
            from: '/knowledge',
          },
          {
            to: '/guides/mysql/generated-columns',
            from: '/knowledge/mysql/generated-columns',
          },
          {
            to: '/guides/postgres/partial-indexes',
            from: '/knowledge/postgres/partial-indexes',
          },
          {
            to: '/guides/postgres/serial-columns',
            from: '/knowledge/postgres/serial-columns',
          },
          {
            to: '/guides/ddl',
            from: '/knowledge/ddl',
          },
          {
            to: '/concepts/url',
            from: '/url',
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
