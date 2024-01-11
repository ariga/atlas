---
id: sqlalchemy
title: Automatic migration planning for SQLAlchemy
slug: /guides/orms/sqlalchemy
---

## TL;DR
* [SQLAlchemy](https://www.sqlalchemy.org) is a Python SQL toolkit and Object Relational Mapper.
* [Atlas](https://atlasgo.io) is an open-source tool for inspecting, planning, linting and
  executing schema changes to your database.
* Developers using SQLAlchemy can use Atlas to automatically plan schema migrations
  for them, based on the desired state of their schema instead of crafting them by hand.

## Automatic migration planning for SQLAlchemy

SQLAlchemy is a popular ORM toolkit widely used in the Python community. SQLAlchemy allows users to
describe their data model using its [declarative-mapping](https://docs.sqlalchemy.org/en/20/orm/declarative_tables.html)
feature. To actually create the underlying tables, users can use the `Base.metadata.create_all` method
which may be sufficient during development where tables can be routinely dropped and re-created.

However, at some point, teams need more control and decide to employ
the [versioned migrations](/concepts/declarative-vs-versioned#versioned-migrations) methodology,
which is a more robust way to manage a database schema.

### Replacing `alembic`

The native way to manage migrations with SQLAlchemy is to use the [Alembic](https://alembic.sqlalchemy.org/en/latest/) migration tool.
Alembic can [automatically generate](https://alembic.sqlalchemy.org/en/latest/autogenerate.html#auto-generating-migrations)
migration scripts from the difference between the current state of the database and the desired state of the application.

A downside of this approach is that in order for it to work, a pre-existing database with the current version of the schema must be connected to.
In many production environments, databases should generally not be reachable from developer workstations,
which means this comparison is normally done against a local copy of the database which may have
undergone some changes that aren't reflected in the existing migrations.
In addition, Alembic auto-generation [fails to detect many kinds of changes](https://alembic.sqlalchemy.org/en/latest/autogenerate.html#what-does-autogenerate-detect-and-what-does-it-not-detect)
and cannot be relied upon to generate production-ready migration scripts without routine manual intervention.

Atlas, on the other hand, can automatically plan database schema migrations for SQLAlchemy
without requiring a connection to such a database and can detect almost any kind of schema change.
Atlas plans migrations by calculating the diff between the _current_ state of the database,
and its _desired_ state.

### How it works

In the context of versioned migrations, the current state can be thought of as the database schema that would have
been created by applying all previous migration scripts.

The desired schema of your application can be provided to Atlas via an [External Schema Datasource](/atlas-schema/projects#data-source-external_schema),
which is any program that can output a SQL schema definition to stdout.

To use Atlas with SQLAlchemy, users can utilize the [SQLAlchemy Atlas Provider](https://github.com/ariga/atlas-provider-sqlalchemy),
which is a small program that can be used to load the schema of a SQLAlchemy project into Atlas.

In this guide, we will show how Atlas can be used to automatically plan schema migrations for
SQLAlchemy users.

## Prerequisites

* A local [SQLAlchemy](https://www.sqlalchemy.org) project.

If you don't have a SQLAlchemy project handy, check out the [SQLAlchemy quick start page](https://docs.sqlalchemy.org/en/20/orm/quickstart.html).

## Using the Atlas SQLAlchemy Provider

In this guide, we will use the [SQLAlchemy Atlas Provider](https://github.com/ariga/atlas-provider-sqlalchemy)
to automatically plan schema migrations for a SQLAlchemy project.

### Installation

Install Atlas from macOS or Linux by running:
```bash
curl -sSf https://atlasgo.sh | sh
```
See [atlasgo.io](https://atlasgo.io/getting-started#installation) for more installation options.

Start Python virtual environment if you haven't already:
```bash
python3 -m venv venv
source venv/bin/activate
```

Install the provider by running:
```bash
pip install atlas-provider-sqlalchemy
``` 

:::info TL;DR

The SQLAlchemy Atlas Provider works by importing your SQLAlchemy models and extracting the schema from them.

Therefore, you will need to run the provider from within your project's Python environment.

:::

### Standalone vs Script mode

The Atlas SQLAlchemy Provider can be used in two modes:

* **Standalone** - If all of your SQLAlchemy models exist in a single module,
 you can use the provider directly to load your SQLAlchemy schema into Atlas.
* **Script** - In other cases, you can use the provider as a python script to load your SQLAlchemy schema into Atlas.

### Standalone mode

In your project directory, create a new file named `atlas.hcl` with the following contents:

```hcl
data "external_schema" "sqlalchemy" {
  program = [
    "atlas-provider-sqlalchemy",
    "--path", "./db",
    "--dialect", "mysql" // mariadb | postgresql | sqlite | mssql
  ]
}

env "sqlalchemy" {
  src = data.external_schema.sqlalchemy.url
  dev = "docker://mysql/8/dev"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
```

#### As Python Script 

If you want to use the provider as a python script, you can use the provider as follows:
Create a new file named `load_models.py` with the following contents:

```python
# import one of the models
from models import User
from atlas_provider_sqlalchemy.ddl import print_ddl

print_ddl("mysql", [User])
```

Next, in your project directory, create a new file named `atlas.hcl` with the following contents:

```hcl
data "external_schema" "sqlalchemy" {
    program = [
        "python3",
        "load_models.py"
    ]
}

env "sqlalchemy" {
  src = data.external_schema.sqlalchemy.url
  dev = "docker://mysql/8/dev"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
```

## Usage

Atlas supports a [versioned migrations](/concepts/declarative-vs-versioned#versioned-migrations)
workflow, where each change to the database is versioned and recorded in a migration file. You can use the
`atlas migrate diff` command to automatically generate a migration file that will migrate the database
from its latest revision to the current SQLAlchemy schema.

Suppose we have the following SQLAlchemy models in our `db/models.py` file:

```python
from typing import List, Optional
from sqlalchemy import ForeignKey, String
from sqlalchemy.orm import Mapped, mapped_column, relationship, DeclarativeBase


class Base(DeclarativeBase):
    pass


class User(Base):
    __tablename__ = "user_account"
    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(30))
    fullname: Mapped[Optional[str]] = mapped_column(String(30))
    addresses: Mapped[List["Address"]] = relationship(
        back_populates="user", cascade="all, delete-orphan"
    )


class Address(Base):
    __tablename__ = "address"
    id: Mapped[int] = mapped_column(primary_key=True)
    email_address: Mapped[str] = mapped_column(String(30))
    user_id: Mapped[int] = mapped_column(ForeignKey("user_account.id"))
    user: Mapped["User"] = relationship(back_populates="addresses")
```

Using the [standalone mode](#standalone-mode) configuration file for the provider,
we can generate a migration file by running this command:

```bash
atlas migrate diff --env sqlalchemy
```

Running this command will generate files similar to this in the `migrations` directory:

```
migrations
|-- 20230918143104.sql
`-- atlas.sum

0 directories, 2 files
```

Examining the contents of `20230918143104.sql`:

```sql
-- Create "user_account" table
CREATE TABLE `user_account` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(30) NOT NULL,
  `fullname` varchar(30) NULL,
  PRIMARY KEY (`id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "address" table
CREATE TABLE `address` (
  `id` int NOT NULL AUTO_INCREMENT,
  `email_address` varchar(30) NOT NULL,
  `user_id` int NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `user_id` (`user_id`),
  CONSTRAINT `address_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `user_account` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

```

Amazing! Atlas automatically generated a migration file that will create the `user_account` and `address` tables in our database.

Next, alter the `user_account` entity to add a new `age` field:

```diff
    fullname: Mapped[Optional[str]] = mapped_column(String(30))
+   age: Mapped[Optional[int]] = mapped_column()
    addresses: Mapped[List["Address"]] = relationship(
        back_populates="user", cascade="all, delete-orphan"
    )
```

Re-run this command:

```bash
atlas migrate diff --env sqlalchemy
```

Observe a new migration file is generated:

```sql
-- Modify "user_account" table
ALTER TABLE `user_account` ADD COLUMN `age` int NULL;
```

## Conclusion

In this guide we demonstrated how projects using SQLAlchemy can use Atlas to automatically
plan schema migrations based only on their data model. To learn more about executing
migrations against your production database, read the documentation for the
[`migrate apply`](/versioned/apply) command.

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT)
