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
                'getting-started/using-the-management-ui'
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
            type: 'doc',
            id: 'cli-reference',
        },
        {
            type: 'doc',
            id: 'ui/atlas-ui-intro',
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
