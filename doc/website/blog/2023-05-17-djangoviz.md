---
title: "Quickly visualize your Django schemas with DjangoViz"
authors: shani-a
tags: [schema, django, visualization, ERD]
---
Having a visual representation of your data model can be helpful as it allows for easier comprehension of complex data
structures, and enables developers to better understand and collaborate on the data model of the application they are
building.

[ER diagrams](https://en.wikipedia.org/wiki/Entity%E2%80%93relationship_model) (ERDs) are a common way to visualize
data models, by showing how data is stored in the database. ERDs are graphical representation of the entities, their
attributes, and the way these entities are related to each other.

Today we are happy to announce the release of [DjangoViz](https://github.com/ariga/djangoviz/), a new tool for
automatically creating ERDs from Django data models.

[Django](https://github.com/django/django) is an open source Python framework for building web applications quickly and
efficiently. In this blog post, I will introduce DjangoViz and demonstrate how to use it for generating Django schema
visualizations using the [Atlas playground](https://gh.atlasgo.cloud/explore/3a5c718d).

[![img](https://atlasgo.io/uploads/images/explore-example.png)](https://gh.atlasgo.cloud/explore/3a5c718d)

### Django ORM

Django ORM is a built-in module in the Django web framework. It offers a high-level abstraction layer that enables
developers to define complex application data models with ease. Unlike traditional ORM frameworks that rely on tables
and foreign keys, Django models are defined using Python objects and relationships:

```python
from django.db import models

class User(models.Model):
    username = models.CharField(max_length=255)
    email = models.EmailField(unique=True)
    password = models.CharField(max_length=255)

class Post(models.Model):
    title = models.CharField(max_length=255)
    content = models.TextField()
    author = models.ForeignKey(User, on_delete=models.CASCADE)
```

When the application runs, Django translates these Python models into database schemas, mapping each model to a
corresponding database table and each field to a corresponding column in the table.   
When working with schemas and making changes to them, being able to understand the full picture just through code can
get complicated very quickly. To help developers better understand their schema, we have created DjangoViz.

### Introducing DjangoViz

For the purpose of this demo, we will follow the Django [Getting started tutorial](https://docs.djangoproject.com/en/4.2/intro/tutorial01/), and showcase how you can use DjangoViz for schema visualization, by visualizing all of Django build-in models that are automatically created when creating a new project.


First, create a new Django project:

```console
django-admin startproject atlas_demo
```

Install the DjangoViz package: 

```console
pip install djangoviz
```

Add DjangoViz to your Django project's INSTALLED_APPS in `settings.py`:

```python
INSTALLED_APPS = [
    ...,
    'djangoviz',
    ...
]
```

DjangoViz support either PostgreSQL or MySQL, in this example I will use PostgreSQL:

Install PostgreSQL driver:

```console
pip install psycopg2-binary
```

Configure the database to work with PostgreSQL in the ‘settings.py` file:


```python
DATABASES = {
   "default": {
       "ENGINE": "django.db.backends.postgresql_psycopg2",
       "NAMEv: "postgresDB",
       'USER': 'postgresUser',
       'PASSWORD': 'postgresP',
       'HOST': '127.0.0.1',
       'PORT': '5455',
   }
}
```

Start a PostgreSQL container:
```console
docker run    -p 5455:5432  -e POSTGRESf_USER=postgresUser   -e POSTGRES_PASSWORD=postgresP  -e POSTGRES_DB=postgresDB   -d   postgres
   }
}
```

Now, you can visualize your schema by running the `djangoviz` management command from your new project directory:

```console
python manage.py djangoviz
```

You will get a public link to your visualization, which will present an ERD and the schema itself in SQL
or [HCL](https://atlasgo.io/guides/ddl#hcl%E2%80%9D%20with%20%E2%80%9Chttps://atlasgo.io/atlas-schema/sql-resources):

```console
Here is a public link to your schema visualization:
  https://gh.atlasgo.cloud/explore/ac523fef
```

When clicking on the link you will see the [ER diagram](https://gh.atlasgo.cloud/explore/ac523fef) of your new project, that includes all of Django build-in models:


[![img](https://atlasgo.io/uploads/images/django-getting-started-schema.png)](https://gh.atlasgo.cloud/explore/ac523fef) . 


### Wrapping up

In this blog post, we discussed DjangoViz, a new tool that helps to quickly visualize Django schemas. With this tool,
you can easily get an overview of the data model, and visual of your schema. We would love to hear your thoughts and
feedback if you decide to give it a go!

Have questions? Feedback? Find our team on [our Discord server](https://discord.gg/zZ6sWVg6NT) :heart:.