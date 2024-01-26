---
id: django
title: Automatic migration planning for Django
slug: /guides/orms/django
---

## TL;DR
* [Django](https://www.djangoproject.com) is a Python web framework, with a built-in ORM.
* [Atlas](https://atlasgo.io) is an open-source tool for inspecting, planning, linting and
  executing schema changes to your database.
* Developers using Django can use Atlas to automatically plan schema migrations
  for them.

## Automatic migration planning for Django
Django is the most popular web framework in the Python community. It includes a [built-in ORM](https://docs.djangoproject.com/en/5.0/#the-model-layer)
which allows users to describe their data model using Python classes. Then create [migrations](https://docs.djangoproject.com/en/5.0/topics/migrations)
using the `python manage.py makemigrations` command, which can be applied to the database using `python manage.py migrate`.

A downside of this approach is that in order to [generate migrations](https://docs.djangoproject.com/en/5.0/ref/django-admin/#django-admin-makemigrations),
a pre-existing database with the current version of the schema must be connected to.
In many production environments, databases should generally not be reachable from developer workstations,
which means this comparison is normally done against a local copy of the database which may have
undergone some changes that aren't reflected in the existing migrations.

Atlas, on the other hand, can automatically plan database schema migrations for Django
without requiring a connection to such a database and can detect almost any kind of schema change.
Atlas plans migrations by calculating the diff between the _current_ state of the database,
and its _desired_ state.

### How it works

In the context of [versioned migrations](/concepts/declarative-vs-versioned#versioned-migrations),
the current state can be thought of as the database schema that would have
been created by applying all previous migration scripts.

The desired schema of your application can be provided to Atlas via an [External Schema Datasource](/atlas-schema/projects#data-source-external_schema),
which is any program that can output a SQL schema definition to stdout.

To use Atlas with Django, users can utilize the [Django Atlas Provider](https://github.com/ariga/atlas-provider-django)
which is a small program that can be used to load the schema of a Django project into Atlas.

In this guide, we will show how Atlas can be used to automatically plan schema migrations for Django users.


## Prerequisites

* A local [Django](https://www.djangoproject.com) project.

If you don't have a Django project handy, check out the [Django getting started page](https://docs.djangoproject.com/en/5.0/intro/tutorial01/)

## Using the Atlas Django Provider

In this guide, we will use the [Django Atlas Provider](https://github.com/ariga/atlas-provider-django)
to automatically plan schema migrations for a Django project.

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
pip install atlas-provider-django
``` 

### Configuration

Add the provider to your Django project's `INSTALLED_APPS` in [`settings.py`](https://docs.djangoproject.com/en/5.0/topics/settings/):

```python
INSTALLED_APPS = [
    ...,
    'atlas_provider_django',
    ...
]
```

In your project directory, where [`manage.py`](https://docs.djangoproject.com/en/5.0/ref/django-admin/) file is located, 
create a new file named `atlas.hcl` with the following contents:

```hcl
data "external_schema" "django" {
  program = [
    "python",
    "manage.py",
    "atlas-provider-django",
    "--dialect", "mysql" // mariadb | postgresql | sqlite | mssql
    
    // if you want to only load a subset of your app models, you can specify the apps by adding
    // "--apps", "app1", "app2", "app3"
  ]
}

env "django" {
  src = data.external_schema.django.url
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
from its latest revision to the current Django schema.

Suppose we have installed [app](https://docs.djangoproject.com/en/5.0/ref/applications/) named `polls`, 
with the following models in our `polls/models.py` file:

```python
from django.db import models


class Question(models.Model):
    question_text = models.CharField(max_length=200)
    pub_date = models.DateTimeField("date published")


class Choice(models.Model):
    question = models.ForeignKey(Question, on_delete=models.CASCADE)
    choice_text = models.CharField(max_length=200)
    votes = models.IntegerField(default=0)
```

We can generate a migration file by running this command:

```bash
atlas migrate diff --env django
```

Running this command will generate files similar to this in the `migrations` directory:

```
migrations
|-- 20240126104629.sql
`-- atlas.sum

0 directories, 2 files
```

Examining the contents of `20240126104629.sql`:

```sql
-- Create "polls_question" table
CREATE TABLE `polls_question` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `question_text` varchar(200) NOT NULL,
  `pub_date` datetime(6) NOT NULL,
  PRIMARY KEY (`id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "polls_choice" table
CREATE TABLE `polls_choice` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `choice_text` varchar(200) NOT NULL,
  `votes` int NOT NULL,
  `question_id` bigint NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `polls_choice_question_id_c5b4b260_fk_polls_question_id` (`question_id`),
  CONSTRAINT `polls_choice_question_id_c5b4b260_fk_polls_question_id` FOREIGN KEY (`question_id`) REFERENCES `polls_question` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

```

Amazing! Atlas automatically generated a migration file that will create the `polls_question` and `polls_choice` tables in our database.
Next, alter the `Question` class to add a new `question_type` field:

```diff
class Question(models.Model):
    question_text = models.CharField(max_length=200)
+   question_type = models.CharField(max_length=20, null=True)
    pub_date = models.DateTimeField("date published")
```
Re-run this command:

```bash
atlas migrate diff --env django
```

Observe a new migration file is generated:

```sql
-- Modify "polls_question" table
ALTER TABLE `polls_question` ADD COLUMN `question_type` varchar(20) NULL;
```

## Conclusion

In this guide we demonstrated how projects using Django can use Atlas to automatically
plan schema migrations based only on their data model. To learn more about executing
migrations against your production database, read the documentation for the
[`migrate apply`](/versioned/apply) command.

Have questions? Feedback? Find our team [on our Discord server](https://discord.gg/zZ6sWVg6NT)
