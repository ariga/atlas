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
            label: 'Atlas Schemas',
            collapsed: false,
            items: [
                {type: 'doc', id: 'atlas-schema/sql-resources', label: 'SQL Resources'},
                {type: 'doc', id: 'atlas-schema/sql-types', label: 'SQL Column Types'},
                {type: 'doc', id: 'atlas-schema/projects', label: 'Project Structure'},
                {type: 'doc', id: 'atlas-schema/input-variables', label: 'Input Variables'},
            ],
        },
        {
            type: 'category',
            label: 'Concepts',
            collapsed: false,
            items: [
                {type: 'doc', id: 'concepts/concepts-url', label: 'URLs'},
                {type: 'doc', id: 'concepts/dev-database', label: 'Dev Database'},
                {type: 'doc', id: 'concepts/ddl', label: 'Data Definition Language'},
            ],
        },
        {
            type: 'category',
            label: 'Integrations',
            collapsed: false,
            items: [
                {type: 'doc', id: 'integrations/terraform-provider', label: 'Terraform Provider'},
                {type: 'doc', id: 'integrations/go-api', label: 'Go API'},
            ]
        },
        {
            type: 'doc',
            id: 'contributing',
        },
        {
            type: 'doc',
            id: 'cli-reference'
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
