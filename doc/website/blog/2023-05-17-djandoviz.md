---
title: "Quickly visualize your Django schemas with DjangoViz"
authors: shani-a
tags: [schema, django, visualization, ERD]
---
Having a visual representation of your data model can be helpful, it allows for easier comprehension of complex data structures, and enables developers to better understand and collaborate on the data model of the application they are building.  

ER diagrams (ERDs) are a common way to visualize data models, by showing how data is stored in the database. ERDs are graphical representation of the entities, their attributes, and the way these entities are related to each other.  

Today we are happy to announce the release of [DjangoViz](https://github.com/ariga/djangoviz/), a new tool for automatically creating ERDs from Django data models.  


[Django](https://github.com/django/django) is an open source Python framework for building web applications quickly and efficiently. In this blog post, I will introduce DjangoViz and demonstrate how to use it for generating Django schema visualizations using the [Atlas playground](https://gh.atlasgo.cloud/explore/3a5c718d).  


[![img](https://atlasgo.io/uploads/images/explore-example.png)](https://gh.atlasgo.cloud/explore/3a5c718d)


### Django ORM 

Django ORM is a built-in module in the Django web framework. It offers a high-level abstraction layer that enables developers to define complex application data models with ease. Unlike traditional ORM frameworks that rely on tables and foreign keys, Django models are defined using Python objects and relationships:  


```python

Code exmple of schema defined in Django ORM



```
  
  
When the application runs, Django translates these Python models into database schemas, mapping each model to a corresponding database table and each field to a corresponding column in the table.   
When working with schemas and making changes to them, being able to understand the full picture just through code can get complicated very quickly. To help developers better understand their schema, we have created DjangoViz.


### Introducing DjangoViz​


I will use the [Pennersr/django-allauth](https://github.com/pennersr/django-allauth/tree/master) project to show how you can use DjangoViz for schema visualization. To follow along, fork the pennersr/django-allauth repo.  


Install the DjangoViz package:

```python
pip install djangoviz
```

Add DjangoViz to your Django project's INSTALLED_APPS in 'settings.py':
```python
INSTALLED_APPS = [
    ...,
    'djangoviz',
    ...
]
```
Now, you can visualize your schema by running the djangoviz management command:  

```python
python manage.py djangoviz
```

You will get a public link to your visualization, which will present an ERD and the schema itself in SQL or [HCL](https://atlasgo.io/guides/ddl#hcl%E2%80%9D%20with%20%E2%80%9Chttps://atlasgo.io/atlas-schema/sql-resources):

```python
Here is a public link to your schema visualization:
       https://gh.atlasgo.cloud/explore/9fe81b90
```


  

When clicking on the link you will see the [ER diagram](https://gh.atlasgo.cloud/explore/9fe81b90) of the Pennersr/django-allauth project:

[![img](https://atlasgo.io/uploads/images/djangoviz-example.png)](https://gh.atlasgo.cloud/explore/9fe81b90)

  

### Wrapping up​
In this blog post, we discussed DjangoViz, a new tool that helps to quickly visualize Django schemas. With this tool, you can easily get an overview of the data model, and visual  of your schema. We would love to hear your thoughts and feedback if you decide to give it a go!



Have questions? Feedback? Find our team on [our Discord server](https://discord.gg/zZ6sWVg6NT) :heart:.