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
                {type: 'doc', id: 'declarative/diff', label: 'Comparing Schemas'},
            ]
        },
        {
            type: "category",
            label: "Versioned Workflows",
            collapsed: false,
            items: [
                {type: 'doc', id: 'versioned/diff', label: 'Generating Migrations'},
                {type: 'doc', id: 'versioned/lint', label: 'Migration Linting'},
                {type: 'doc', id: 'versioned/new', label: 'Manual Migrations'},
                {type: 'doc', id: 'versioned/apply', label: 'Applying Migrations'},
                {type: 'doc', id: 'versioned/troubleshoot', label: 'Migration Troubleshooting'},
                {type: 'doc', id: 'versioned/import', label: 'Import Existing Migrations'},
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
                {type: 'doc', id: 'concepts/concepts-url', label: 'URLs'},
                {type: 'doc', id: 'concepts/dev-database', label: 'Dev Database'},
                {type: 'doc', id: 'concepts/migration-directory-integrity', label: 'Directory Integrity'},
                {type: 'doc', id: 'concepts/workflows', label: 'Declarative vs Versioned'},
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
            label: 'Deploying schema migrations',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/deploying/intro',
                    label: 'Introduction'
                },
                {
                    type: 'doc',
                    id: 'guides/deploying/image',
                    label: 'Creating container images'
                },
                {
                    type: 'doc',
                    id: 'guides/deploying/aws-ecs-fargate',
                    label: 'AWS ECS (Fargate)'
                },
                {
                    type: 'doc',
                    id: 'guides/deploying/helm',
                    label: 'Kubernetes (Helm)'
                },
                {
                    type: 'doc',
                    id: 'guides/deploying/k8s-init-containers',
                    label: 'Kubernetes (Init Container)'
                },
                {
                    type: 'doc',
                    id: 'guides/deploying/fly-io',
                    label: 'Fly.io'
                }
            ]
        },
        {
            type: 'category',
            label: 'Testing',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/testing/integration-docker-compose',
                    label: 'With docker-compose'
                },
                {
                    type: 'doc',
                    id: 'guides/testing/testing-github-actions',
                    label: 'With Github Actions'
                }
            ]
        },
        {
            type: 'category',
            label: 'MySQL',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/mysql/descending-indexes',
                    label: 'Descending Indexes'
                },
                {
                    type: 'doc',
                    id: 'guides/mysql/functional-indexes',
                    label: 'Functional Indexes'
                },
                {
                    type: 'doc',
                    id: 'guides/mysql/prefix-indexes',
                    label: 'Prefix Indexes'
                },
                {
                    type: 'doc',
                    id: 'guides/mysql/check-constraint',
                    label: 'CHECK Constraint'
                },
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
                {
                    type: 'doc',
                    id: 'guides/mysql/mysql-my-102',
                    label: 'Inline REFERENCES (MY102)'
                },
                {
                    type: 'doc',
                    id: 'guides/mysql/mysql-ds-103',
                    label: 'Column Drop (DS103)'
                },
                {
                    type: 'doc',
                    id: 'guides/mysql/mysql-cd-101',
                    label: 'Constraint Drop (CD101)'
                },
            ],
        },
        {
            type: 'category',
            label: 'Terraform',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/terraform/named-databases',
                    label: 'Named Databases'
                }
            ]
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
                {
                    type: 'doc',
                    id: 'guides/postgres/index-operator-classes',
                    label: 'Index Operator Classes'
                },
                {
                    type: 'doc',
                    id: 'guides/postgres/ar-101',
                    label: 'Optimal data alignment (AR101)'
                },
                {
                    type: 'doc',
                    id: 'guides/postgres/descending-indexes',
                    label: 'Descending Indexes'
                },
                {
                    type: 'doc',
                    id: 'guides/postgres/functional-indexes',
                    label: 'Functional Indexes'
                }
            ],
        },
        {
            type: 'category',
            label: 'SQLite',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/sqlite/partial-indexes',
                    label: 'Partial Indexes'
                },
                {
                    type: 'doc',
                    id: 'guides/sqlite/descending-indexes',
                    label: 'Descending Indexes'
                },
                {
                    type: 'doc',
                    id: 'guides/sqlite/functional-indexes',
                    label: 'Functional Indexes'
                }
            ],
        },
        {
            type: 'category',
            label: 'Migration tools',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/migration-tools/golang-migrate',
                    label: 'golang-migrate'
                },
                {
                    type: 'doc',
                    id: 'guides/migration-tools/goose-import',
                    label: 'Importing from goose'
                }
            ]
        },
        {
            type: 'category',
            label: 'Frameworks',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/orms/gorm',
                    label: 'GORM'
                },
                {
                    type: 'doc',
                    id: 'guides/frameworks/sqlc-declarative',
                    label: 'Declarative migrations for sqlc'
                },
                {
                    type: 'doc',
                    id: 'guides/frameworks/sqlc-versioned',
                    label: 'Versioned migrations for sqlc'
                }
            ]
        },
        {
            type: 'category',
            label: 'Cloud',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/cloud/explore-inspection',
                    label: 'How to inspect a local database in the Cloud'
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
