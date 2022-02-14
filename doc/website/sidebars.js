/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */

module.exports = {
    // By default, Docusaurus generates a sidebar from the docs folder structure
    tutorialSidebar: [
        {
            type: 'category',
            label: 'Getting Started',
            items: [
                'getting-started/getting-started-installation',
                'getting-started/getting-started-inspection',
                'getting-started/getting-started-apply',
                'getting-started/getting-started-management-ui',
            ]
        },
        {
            type: 'category',
            label: 'Data Definition Language',
            items: [
                {type: 'doc', id: 'ddl/ddl-intro', label: 'Introduction'},
                {type: 'doc', id: 'ddl/ddl-sql', label: 'SQL'},
                {type: 'doc', id: 'ddl/ddl-sql-types', label: 'SQL Types'},
            ]
        },
        {
            type: 'category',
            label: 'CLI',
            items: [
                {type: 'doc', id: 'cli/cli-reference', label: 'Reference'},
                {type: 'doc', id: 'cli/cli-dsn', label: 'DSN'},
            ]
        },
        {
            type: 'doc',
            id: 'ui/atlas-ui-intro',
        },
        {
            type: 'category',
            label: 'Go API',
            items: [
                {type: 'doc', id: 'go-api/intro', label: 'Introduction'},
                {type: 'doc', id: 'go-api/inspect', label: 'Inspecting Schemas'},
            ]
        },
        {
            type: 'doc',
            id: 'contributing',
        }
    ],
    aboutSidebar: [
        {
            type: 'doc',
            label: 'About',
            id: 'about',
        }
    ]
    // But you can create a sidebar manually
    /*
    tutorialSidebar: [
      {
        type: 'category',
        label: 'Tutorial',
        items: ['hello'],
      },
    ],
     */
};
