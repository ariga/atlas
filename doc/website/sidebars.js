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
            type: 'doc',
            id: 'home/docs',
            label: 'Home',
        },
        {
            type: 'category',
            label: 'Getting Started',
            collapsed: false,
            items: [
                'getting-started/getting-started',
            ]
        },
        {
            type: 'doc',
            id: 'atlas-vs-others',
            label: 'Atlas vs Others'
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
                {type: 'doc', id: 'versioned/down', label: 'Down Migrations'},
                {type: 'doc', id: 'versioned/checks', label: 'Pre-migration Checks'},
                {type: 'doc', id: 'versioned/new', label: 'Manual Migrations'},
                {type: 'doc', id: 'versioned/troubleshoot', label: 'Migration Troubleshooting'},
                {type: 'doc', id: 'versioned/import', label: 'Import Existing Migrations'},
                {type: 'doc', id: 'versioned/checkpoint', label: 'Checkpoints'},
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
            type: "category",
            label: "Testing Framework",
            collapsed: false,
            items: [
                {type: 'doc', id: 'testing/schema', label: 'Testing Schemas'},
                {type: 'doc', id: 'testing/migrate', label: 'Testing Migrations'},
            ]
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
                {
                    type: 'category',
                    label: 'Features',
                    collapsed: true,
                    items: [
                        {type: 'doc', id: 'cloud/features/registry', label: 'Schema Registry'},
                        {type: 'doc', id: 'cloud/features/schema-docs', label: 'Schema Docs'},
                        {type: 'doc', id: 'cloud/features/pre-migration-checks', label: 'Pre-migration Checks'},
                        {type: 'doc', id: 'cloud/features/troubleshooting', label: 'Migration Troubleshooting'},
                        {type: 'doc', id: 'cloud/features/monitoring', label: 'Schema Monitoring'},
                    ],
                },
                {type: 'doc', id: 'cloud/pricing', label: 'Pricing'},
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
                {type: 'doc', id: 'integrations/circleci-orbs', label: 'CircleCI Orbs'},
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
                {
                    type: 'doc',
                    id: 'cli-data-privacy',
                    label: 'Data Privacy'
                }
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
            label: 'Database-per-Tenant',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    label: 'Introduction',
                    id: 'guides/database-per-tenant/intro',
                },
                {
                    type: 'doc',
                    id: 'guides/database-per-tenant/target-groups',
                    label: 'Target Groups'
                },
                {
                    type: 'doc',
                    id: 'guides/database-per-tenant/deploying',
                    label: 'Deploying'
                },
                {
                    type: 'doc',
                    id: 'guides/database-per-tenant/control-plane',
                    label: 'Control Plane'
                }
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
                },
                {
                    type: 'doc',
                    id: 'guides/deploying/k8s-operator-certs',
                    label: 'SSL Certs (Kubernetes)'
                },
            ]
        },
        {
            type: 'doc',
            id: 'guides/atlas-in-docker',
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
                },
                {
                    type: 'doc',
                    id: 'guides/testing/testing-views',
                    label: 'Testing Views'
                },
                {
                    type: 'doc',
                    id: 'guides/testing/testing-functions',
                    label: 'Testing Functions'
                },
                {
                    type: 'doc',
                    id: 'guides/testing/testing-domains',
                    label: 'Testing Domains'
                },
                {
                    type: 'doc',
                    id: 'guides/testing/testing-procedures',
                    label: 'Testing Stored Procedures'
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
                    type: 'category',
                    label: 'GORM',
                    link: {
                        type: 'doc',
                        id: 'guides/orms/gorm',
                    },
                    items: [
                        {type: 'doc', id: 'guides/orms/gorm/composite-types', label: 'Composite Types'},
                        {type: 'doc', id: 'guides/orms/gorm/domain-types', label: 'Domain Types'},
                        {type: 'doc', id: 'guides/orms/gorm/enum-types', label: 'Enum Types'},
                        {type: 'doc', id: 'guides/orms/gorm/extensions', label: 'Extensions'},
                        {type: 'doc', id: 'guides/orms/gorm/rls', label: 'Row-Level Security'},
                        {type: 'doc', id: 'guides/orms/gorm/triggers', label: 'Triggers'},
                    ]
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
                    type: 'category',
                    label: 'Sequelize',
                    link: {
                        type: 'doc',
                        id: 'guides/orms/sequelize',
                    },
                    items: [
                        {type: 'doc', id: 'guides/orms/sequelize/composite-types', label: 'Composite Types'},
                        {type: 'doc', id: 'guides/orms/sequelize/domain-types', label: 'Domain Types'},
                        {type: 'doc', id: 'guides/orms/sequelize/rls', label: 'Row-Level Security'},
                        {type: 'doc', id: 'guides/orms/sequelize/triggers', label: 'Triggers'},
                    ],
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
                    id: 'guides/orms/django',
                    label: 'Django'
                },
                {
                    type: 'doc',
                    id: 'guides/orms/doctrine',
                    label: 'Doctrine'
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
                    id: 'guides/postgres/postgres-automatic-migrations',
                    label: 'Automatic Migration Planning'
                },
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
                    id: 'guides/postgres/pg-110',
                    label: 'Optimal data alignment (PG110)'
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
            id: 'guides/getting-started-mssql',
            label: 'SQL Server'
        },
        {
            type: 'doc',
            id: 'guides/getting-started-redshift',
            label: 'Redshift'
        },
        {
            type: 'category',
            label: 'Archive',
            collapsed: false,
            items: [
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
            ]
        }
    ],
    eval: [
        {
            type: 'category',
            label: 'Evaluating Atlas',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/evaluation/intro'
                },
            ]
        },
        {
            type: 'category',
            label: 'Installation',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/evaluation/install'
                },
                {
                    type: 'doc',
                    id: 'guides/evaluation/connect'
                },
                {
                    type: 'doc',
                    id: 'guides/evaluation/verify-atlas',
                    label: 'Verify Atlas'
                }
            ]
        },
        {
            type: 'category',
            label: 'Setting Up',
            collapsed: false,
            items: [
                {
                    type: 'doc',
                    id: 'guides/evaluation/project-structure'
                },
                {
                    type: 'doc',
                    id: 'guides/evaluation/schema-as-code'
                },
                {
                    type: 'doc',
                    id: 'guides/evaluation/setup-migrations',
                    label: 'Migration Directory'
                }
            ]
        },
    ]
};
