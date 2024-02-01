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
                {type: 'doc', id: 'versioned/intro', label: 'Quick Introduction'},
                {type: 'doc', id: 'versioned/diff', label: 'Generating Migrations'},
                {
                    type: 'category', label: 'Migration Linting',
                    link: {
                        type: 'doc',
                        id: 'versioned/lint',
                    },
                    items: [
                        {type: 'doc', id: 'lint/analyzers', label: 'Analyzers and Checks'},
                    ]
                },
                {type: 'doc', id: 'versioned/apply', label: 'Applying Migrations'},
                {type: 'doc', id: 'versioned/checks', label: 'Pre-migration Checks'},
                {type: 'doc', id: 'versioned/new', label: 'Manual Migrations'},
                {type: 'doc', id: 'versioned/troubleshoot', label: 'Migration Troubleshooting'},
                {type: 'doc', id: 'versioned/import', label: 'Import Existing Migrations'},
            ]
        },
        {
            type: 'category',
            label: 'Atlas Schema',
            collapsed: false,
            items: [
                {
                    type: 'category', label: 'HCL Syntax',
                    link: {
                        type: 'doc',
                        id: 'atlas-schema/hcl-schema',
                    },
                    items: [
                        {type: 'doc', id: 'atlas-schema/hcl-types', label: 'Column Types'},
                        {type: 'doc', id: 'atlas-schema/hcl-variables', label: 'Input Variables'},
                    ]
                },
                {type: 'doc', id: 'atlas-schema/sql-schema', label: 'SQL Syntax'},
                {type: 'doc', id: 'atlas-schema/external-schema', label: 'External Integrations'},
                {type: 'doc', id: 'atlas-schema/projects', label: 'Project Configuration'},
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
                {type: 'doc', id: 'cloud/directories', label: 'Connect Directories'},
                {type: 'doc', id: 'cloud/setup-ci', label: 'CI Setup'},
                {type: 'doc', id: 'cloud/bots', label: 'Creating Bots'},
                {type: 'doc', id: 'cloud/deployment', label: 'Deployments'},
                {type: 'doc', id: 'cloud/agents', label: 'Drift Detection'},
            ],
        },
        {
            type: 'category',
            label: 'Integrations',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    label: 'Kubernetes Operator',
                    id: 'integrations/kubernetes/operator'
                },
                {type: 'doc', id: 'integrations/github-actions', label: 'GitHub Actions'},
                {type: 'doc', id: 'integrations/terraform-provider', label: 'Terraform Provider'},
                {type: 'doc', id: 'integrations/go-sdk', label: 'Go SDK'},
            ]
        },
        {
            type: 'doc',
            id: 'contributing',
        },
        {
            type: 'category',
            label: 'CLI Reference',
            collapsed: false,
            link: {
              type: 'doc',
              id: 'cli-reference'
            },
            items: [
                {
                    type: 'doc',
                    id: 'features'
                },
                {
                    type: 'doc',
                    id: 'community-edition'
                },
            ]
        },
        {
            type: 'doc',
            id: 'support',
        }
    ],
    guides: [
        {
            type: 'doc',
            id: 'guides/guides'
        },
        {
          type: 'category',
          label: 'For Platform Teams',
          collapsed: false,
            items: [
                {
                    type: 'doc',
                    label: 'Modern Database CI/CD',
                    id: 'guides/modern-database-ci-cd',
                },
            ]
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
                    id: 'guides/deploying/cloud-dir',
                    label: 'Deploying from Atlas Cloud'
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
                    id: 'guides/deploying/k8s-argo',
                    label: 'Kubernetes (Argo CD)'
                },
                {
                    type: 'doc',
                    id: 'guides/deploying/k8s-flux',
                    label: 'Kubernetes (Flux CD)'
                },
                {
                    type: 'doc',
                    id: 'guides/deploying/fly-io',
                    label: 'Fly.io'
                },
                {
                    type: 'doc',
                    id: 'guides/deploying/cloud-sql-via-github-actions',
                    label: 'GCP CloudSQL (GH Actions)'
                },
                {
                    type: 'doc',
                    id: 'guides/deploying/secrets',
                    label: 'Working with secrets'
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
                    label: 'With GitHub Actions'
                }
            ]
        },
        {
            type: 'category',
            label: 'CI Platforms',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    label: 'GitLab',
                    id: 'guides/ci-platforms/gitlab',
                }
            ]
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
            label: 'Migration directories',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/migration-dirs/template-directory',
                    label: 'Template directories'
                },
            ]
        },
        {
            type: 'category',
            label: 'Migration tools',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/migration-tools/golang-migrate',
                    label: 'Working with golang-migrate'
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
                    id: 'guides/orms/beego',
                    label: 'beego'
                },
                {
                    type: 'doc',
                    id: 'guides/orms/hibernate',
                    label: 'Hibernate'
                },
                {
                    type: 'doc',
                    id: 'guides/orms/sequelize',
                    label: 'Sequelize'
                },
                {
                    type: 'doc',
                    id: 'guides/orms/typeorm',
                    label: 'TypeORM'
                },
                {
                    type: 'doc',
                    id: 'guides/orms/sqlalchemy',
                    label: 'SQLAlchemy'
                },
                {
                    type: 'doc',
                    id: 'guides/frameworks/sqlc-declarative',
                    label: 'sqlc (declarative)'
                },
                {
                    type: 'doc',
                    id: 'guides/frameworks/sqlc-versioned',
                    label: 'sqlc (versioned)'
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
                },
                {
                    type: 'doc',
                    id: 'guides/sqlite/turso',
                    label: 'Working with Turso'
                }
            ],
        },
        {
            type: 'doc',
            id: 'guides/getting-started-clickhouse',
            label: 'ClickHouse'
        },
        {
            type: 'doc',
            id: 'guides/ddl'
        },
    ]
};
