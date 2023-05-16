---
title: "Announcing the Atlas HCL VSCode Extension"
authors: datdao
tags: [vscode, hcl]
---

## Introduction

Atlas supports many ways to describe database schemas. Using [Schema Loaders](https://atlasgo.io/blog/2023/01/05/atlas-v090#schema-loaders),
Atlas gives users the flexibility to define schemas using plain SQL, a connection to another database or using 
[Atlas HCL](https://atlasgo.io/atlas-schema/sql-resources). 

## What is Atlas HCL?

Atlas includes a data-definition language based on a Terraform-like [syntax](https://atlasgo.io/guides/ddl#hcl). This
language, which was designed specifically for describing database schemas is based on HashiCorp's [HCL](https://github.com/hashicorp/hcl)
toolkit for creating structured configuration languages.

This data definition language is easy to read, edit, and visualize, and it allows us to organize our schema data in a
efficient way that has many benefits to using plain SQL.

## What is the Atlas HCL VSCode Extension?

To make it easier for users to define schemas using Atlas HCL, we're happy to announce the release of the official
[Atlas HCL VSCode Extension](https://marketplace.visualstudio.com/items?itemName=Ariga.atlas-hcl). 
This extension provides a rich set of features that will help users define schemas using Atlas HCL.

In this post, we'll take a closer look at the features of this extension
and how it can benefit your work.

## Features

### Auto Completion

The extension provides auto completion support for all SQL resources available within Atlas. For a comprehensive
list of the supported resources, please visit this
document: [SQL Resources](https://atlasgo.io/atlas-schema/sql-resources).

![Completion](https://atlasgo.io/uploads/images/vscode/vscode-completion.gif)

![Batch Completion](https://atlasgo.io/uploads/images/vscode/vscode-batch-completion.gif)

### Referencing Qualified Tables

This refers to a feature that allows for referencing tables and their columns in a structured manner, either relative to
the current location (e.g. **_column.id_**) or absolute to the table (e.g. **_table.user.column.id_**).

![Referencing Qualified Tables](https://atlasgo.io/uploads/images/vscode/vscode-references.gif)

### Linting

To ensure that errors are caught from the beginning, the Atlas HCL VSCode Extension offers a range of validations that
enhance type safety for users. The specific validations provided will vary depending on the SQL dialect being used. This
feature is designed to help users identify and fix issues early on, resulting in a more efficient and effective
schema-building process.

![Linting](https://atlasgo.io/uploads/images/vscode/vscode-validation.png)

## Usage

To use this extension, you can either install it directly from the VSCode desktop or download it from
the [Marketplace](https://marketplace.visualstudio.com/items?itemName=Ariga.atlas-hcl). The extension will appear in the
extensions list as **Atlas HCL**.

![Extension](https://atlasgo.io/uploads/images/vscode/vscode-extension.png)

After installing the Atlas HCL extension, it automatically registers for all the languages that Atlas supports. These
languages include:

| Name              | Language ID             | Description           
|-------------------|-------------------------|-----------------------|
| Atlas SQLite      | atlas-schema-sqlite     | SQLite dialect        
| Atlas MySQL       | atlas-schema-mysql      | MySQL dialect         
| Atlas PostgresQL  | atlas-schema-postgresql | PostgresQL dialect    
| Atlas MariaDB     | atlas-schema-mariadb    | MariaDB dialect       
| Atlas Config File | atlas-config            | Project Configuration 

By default, the extension will auto-detect atlas.hcl files in your project.
However, if you have any schema files in your project, you must choose the language for them. You can do this in three
ways in vscode:

1. Click on "Change Language" for your schema files in this section at bottom. (See image below)

![Installation](https://atlasgo.io/uploads/images/vscode/vscode-installation.png)

2. Use the Command Palette and choose "Change language mode" (See image below)

![Installation by command palette](https://atlasgo.io/uploads/images/vscode/vscode-installation-2.png)

3. Use [Language Identifiers](https://code.visualstudio.com/docs/languages/identifiers)

## Wrapping Up

In this post, we've introduced **The Atlas HCL Extension**. There's much more inside, and we invite you to install it
today and give it a try.

## What next?

* Follow the [Getting Started](https://atlasgo.io/cli/getting-started/setting-up) guide.
* Join our [Discord Server](https://discord.gg/zZ6sWVg6NT).
* Follow us [on Twitter](https://twitter.com/ariga_io).
