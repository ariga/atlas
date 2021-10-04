/** @type {import('@docusaurus/types').DocusaurusConfig} */
module.exports = {
  title: 'Atlas',
  tagline: 'Manage your data',
  url: 'https://atlasgo.io',
  baseUrl: '/',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  favicon: 'img/favicon.ico',
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
        alt: 'My Site Logo',
        src: 'img/square.svg',
      },
      items: [
        {
          type: 'doc',
          docId: 'getting-started',
          position: 'left',
          label: 'Welcome',
        },
        // {to: '/blog', label: 'Blog', position: 'left'},
        {
          href: 'https://github.com/ariga/atlas',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {
              label: 'Welcome',
              to: '/',
            },
          ],
        },
        {
          title: 'Community',
          items: [
            // {
            //   label: 'Stack Overflow',
            //   href: 'https://stackoverflow.com/questions/tagged/docusaurus',
            // },
            // {
            //   label: 'Discord',
            //   href: 'https://discordapp.com/invite/docusaurus',
            // },
            // {
            //   label: 'Twitter',
            //   href: 'https://twitter.com/docusaurus',
            // },
          ],
        },
        {
          title: 'More',
          items: [
            // {
            //   label: 'Blog',
            //   to: '/blog',
            // },
            {
              label: 'GitHub',
              href: 'https://github.com/ariga/atlas',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} The Atlas Authors, Inc. Built with Docusaurus.`,
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
