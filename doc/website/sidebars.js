/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */

module.exports = {
    documentation: [
        {
            type: 'category',
            label: 'Getting Started',
            collapsed: false,
            items: [
                'getting-started/getting-started-installation',
                'getting-started/getting-started-inspection',
                'getting-started/getting-started-apply',
            ]
        },
        {
            type: 'category',
            label: 'Data Definition Language',
            collapsed: false,
            items: [
                'ddl/ddl-sql',
                'ddl/ddl-sql-types',
                'ddl/ddl-input-variables',
                'ddl/ddl-intro',
            ],
        },
        {
            type: 'doc',
            id: 'dev-database'
        },
        {
            type: 'category',
            label: 'CLI',
            items: [
                {type: 'doc', id: 'cli/cli-reference', label: 'Reference'},
                {type: 'doc', id: 'cli/cli-url', label: 'URLs'},
                {type: 'doc', id: 'cli/projects'},
            ]
        },
        {
            type: 'doc',
            id: 'terraform-provider',
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
    knowledge: [
        {
            type: 'doc',
            id: 'knowledge/knowledge'
        },
        {
            type: 'category',
            label: 'MySQL',
            collapsed: false,
            items: [
                {
                  type: 'doc',
                  id: 'knowledge/mysql/generated-columns',
                  label: 'Generated Columns'
                },
            ],
        },
        {
            type: 'category',
            label: 'PostgreSQL',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'knowledge/postgres/serial-columns',
                    label: 'Serial Type Columns'
                },
            ],
        },
    ],
    about: [
        {
            type: 'doc',
            label: 'About',
            id: 'about',
        }
    ]
};
