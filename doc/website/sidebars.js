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
                'getting-started/getting-started',
            ]
        },
        {
            type: "category",
            label: "Declarative Workflows",
            collapsed: false,
            items: [
                {type: 'doc', id: 'declarative/inspect', label: 'Schema Inspection'},
                {type: 'doc', id: 'declarative/apply', label: 'Applying Changes'},
                {type: 'doc', id: 'declarative/diff', label: 'Calculating Diffs'},
            ]
        },
        {
            type: "category",
            label: "Versioned Workflows",
            collapsed: false,
            items: [
                {type: 'doc', id: 'versioned/diff', label: 'Migration Authoring'},
                {type: 'doc', id: 'versioned/lint', label: 'Migration Linting'},
                {type: 'doc', id: 'versioned/new', label: 'Manual Migrations'},
                {type: 'doc', id: 'versioned/apply', label: 'Migration Applying'},
                {type: 'doc', id: 'versioned/troubleshoot', label: 'Migration Troubleshooting'},
                {type: 'doc', id: 'versioned/import', label: 'Migration Import'},
            ]
        },
        {
            type: 'category',
            label: 'Atlas Schemas',
            collapsed: false,
            items: [
                {type: 'doc', id: 'atlas-schema/sql-resources', label: 'SQL Resources'},
                {type: 'doc', id: 'atlas-schema/sql-types', label: 'SQL Column Types'},
                {type: 'doc', id: 'atlas-schema/projects', label: 'Project Configuration'},
                {type: 'doc', id: 'atlas-schema/input-variables', label: 'Input Variables'},
            ],
        },
        {
            type: 'category',
            label: 'Concepts',
            collapsed: false,
            items: [
                {type: 'doc', id: 'concepts/workflows', label: 'Declarative vs Versioned'},
                {type: 'doc', id: 'concepts/concepts-url', label: 'URLs'},
                {type: 'doc', id: 'concepts/dev-database', label: 'Dev Database'},
                {type: 'doc', id: 'concepts/migration-directory-integrity', label: 'Directory Integrity'},
            ],
        },
        {
            type: 'category',
            label: 'Cloud',
            collapsed: false,
            items: [
                {type: 'doc', id: 'cloud/getting-started', label: 'Getting Started'},
            ],
        },
        {
            type: 'category',
            label: 'Integrations',
            collapsed: false,
            items: [
                {type: 'doc', id: 'integrations/github-actions', label: 'GitHub Actions'},
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
    guides: [
        {
            type: 'doc',
            id: 'guides/guides'
        },
        {
            type: 'category',
            label: 'MySQL',
            collapsed: false,
            items: [
                {
                  type: 'doc',
                  id: 'guides/mysql/generated-columns',
                  label: 'Generated Columns'
                },
                {
                    type: 'doc',
                    id: 'guides/mysql/terraform-mysql-rds',
                    label: 'Managing with Terraform'
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
                    id: 'guides/postgres/serial-columns',
                    label: 'Serial Type Columns'
                },
                {
                    type: 'doc',
                    id: 'guides/postgres/partial-indexes',
                    label: 'Partial Indexes'
                },
                {
                    type: 'doc',
                    id: 'guides/postgres/included-columns',
                    label: 'Covering Indexes'
                },
            ],
        },
        {
            type: 'category',
            label: 'Migration tools',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id:'guides/migration-tools/golang-migrate',
                    label: 'golang-migrate'
                }
            ]
        },
        {
            type: 'category',
            label: 'ORMs',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id:'guides/orms/gorm',
                    label: 'GORM'
                }
            ]
        },
        {
            type: 'doc',
            id: 'guides/ddl'
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
