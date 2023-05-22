---
title: "Quickly visualize your Django schemas with DjangoViz"
authors: shani-a
tags: [schema, django, visualization, ERD]
---
Having a visual representation of your data model can be helpful as it allows for easier comprehension of complex data
structures, and enables developers to better understand and collaborate on the data model of the application they are
building.

[Entity relationship diagrams](https://en.wikipedia.org/wiki/Entity%E2%80%93relationship_model) (ERDs) are a common way to visualize
data models, by showing how data is stored in the database. ERDs are graphical representations of the entities, their
attributes, and the way these entities are related to each other.

Today we are happy to announce the release of [DjangoViz](https://github.com/ariga/djangoviz/), a new tool for
automatically creating ERDs from Django data models.

[Django](https://github.com/django/django) is an open source Python framework for building web applications quickly and
efficiently. In this blog post, I will introduce DjangoViz and demonstrate how to use it for generating Django schema
visualizations using the [Atlas playground](https://gh.atlasgo.cloud/explore/3a5c718d).

[![img](https://atlasgo.io/uploads/images/explore-example.png)](https://gh.atlasgo.cloud/explore/3a5c718d)

### Django ORM

[Django ORM](https://docs.djangoproject.com/en/4.2/topics/db/models/) is a built-in module in the Django web framework. It offers a high-level abstraction layer that enables
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

For the purpose of this demo, we will follow the Django [getting started tutorial](https://docs.djangoproject.com/en/4.2/intro/tutorial01/), 
and showcase how you can use DjangoViz to visualize the default models included by Django's [`startproject`](https://docs.djangoproject.com/en/4.2/ref/django-admin/#startproject) command.


First, install Django and create a new project:

```console
pip install Django
django-admin startproject atlas_demo
cd atlas_demo
```

Install the DjangoViz package: 

```console
pip install djangoviz
```

Add DjangoViz to your Django project's INSTALLED_APPS in `atlas_demo/settings.py`:

```python
INSTALLED_APPS = [
    ...,
    'djangoviz',
    ...
]
```

DjangoViz supports either PostgreSQL or MySQL, in this example we will use PostgreSQL:

Install the PostgreSQL driver:

```console
pip install psycopg2-binary
```

Configure the database to work with PostgreSQL in the `settings.py` file:


```python
DATABASES = {
   "default": {
       "ENGINE": "django.db.backends.postgresql_psycopg2",
       "NAME": "postgres",
       "USER": "postgres",
       "PASSWORD": "pass",
       "HOST": "127.0.0.1",
       "PORT": "5432",
   }
}
```

Start a PostgreSQL container:
```console
docker run --rm -p 5432:5432  -e POSTGRES_PASSWORD=pass -d postgres:15
```

Now, you can visualize your schema by running the `djangoviz` management command from your new project directory:

```console
python manage.py djangoviz
```

You will get a public link to your visualization, which will present an ERD and the schema itself in SQL
or [HCL](https://atlasgo.io/atlas-schema/sql-resources):

```console
Here is a public link to your schema visualization:
  https://gh.atlasgo.cloud/explore/ac523fef
```

When clicking on the link you will see the [ERD](https://gh.atlasgo.cloud/explore/ac523fef) of your new project:


[![img](https://atlasgo.io/uploads/images/django-getting-started-schema.png)](https://gh.atlasgo.cloud/explore/ac523fef) 


### Wrapping up

In this post, we discussed DjangoViz, a new tool that helps to quickly visualize Django schemas. With this tool,
you can easily get an overview of the data model and visual of your schema. We would love to hear your thoughts and
feedback if you decide to give it a go!

Have questions? Feedback? Find our team on [our Discord server](https://discord.gg/zZ6sWVg6NT) :heart:.