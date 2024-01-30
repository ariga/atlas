import React from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import Heading from '@theme/Heading';
import Image from '@theme/IdealImage';

export enum CardImage {
    Action = "action",
    CI = "ci",
    ClickHouse = "clickhouse",
    Cloud = "cloud",
    Config = "config",
    DataSource = "datasource",
    Deployment = "deployment",
    Docker = "docker",
    ECS = "ecs",
    GitHub = "github",
    Helm = "helm",
    Integration = "integration",
    Kubernetes = "kubernetes",
    Migrate = "migrate",
    MySQL = "mysql",
    Operator = "operator",
    Postgres = "postgres",
    Schema = "schema",
    SQLite = "sqlite",
    Terraform = "terraform",
    Testing = "testing",
    Tools = "tools",
}

interface CardProps {
    name: string;
    image: CardImage;
    url: string;
    description: JSX.Element;
}

export function Card({name, image, url, description}: CardProps) {
  return (
    <div className="col col--4 margin-bottom--lg">
      <div className={clsx('card')}>
        <div className={clsx('card__image')}>
          <Link to={url}>
              <Image img={`https://atlasgo.io/uploads/cards/${image}.png`} alt={`${name}'s image`} />
          </Link>
        </div>
        <div className="card__body">
          <Heading as="h3">{name}</Heading>
          <p>{description}</p>
        </div>
        <div className="card__footer">
          <div className="button-group button-group--block">
            <Link className="button button--secondary" to={url}>
              Read guide
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
