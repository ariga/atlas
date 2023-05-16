# Boost Your Productivity with Atlas HCL Extension for Visual Studio Code

## Introduction
As we know, Atlas supports us in defining database schemas using the [HCL Syntax](https://atlasgo.io/guides/ddl#hcl). This data definition language is easy to read, edit, and visualize, and it allows us to organize our schema data in a more efficient way than tedious SQL. Leveraging this feature, our team is excited to introduce you to the release of the public beta of the official[ Atlas HCL VSCode Extension](https://marketplace.visualstudio.com/items?itemName=Ariga.atlas-hcl) that will maximize efficiency when defining schemas using Atlas HCL. We'll take a closer look at the features of this extension and how it can benefit your work.

## Features

### Auto Completion
The extension provides robust auto completion support for all SQL resources available within Atlas. For a comprehensive list of the supported resources, please visit this document: [SQL Resources](https://atlasgo.io/atlas-schema/sql-resources).

<div>
  <img src="assets/auto-completion-1.gif" width="48%">
  <img src="assets/auto-completion-2.gif" width="48%">
</div>


### Referencing Qualified Tables
This refers to a feature that allows for referencing tables and their columns in a structured manner, either relative to the current location (e.g. **_column.id_**) or absolute to the table (e.g. **_table.user.column.id_**).


![Referencing Qualified Tables](assets/reference.gif)


### Linting
To ensure that errors are caught from the beginning, the Atlas HCL VSCode Extension offers a range of validations that enhance type safety for users. The specific validations provided will vary depending on the SQL dialect being used. This feature is designed to help users identify and fix issues early on, resulting in a more efficient and effective schema-building process.

<img src="assets/validation.png">

## Usage

To use this extension, you can either install it directly from the VSCode desktop or download it from the [Marketplace](https://marketplace.visualstudio.com/items?itemName=Ariga.atlas-hcl). The extension will appear in the extensions list as **Atlas HCL**.

<img src="assets/extension.png">

After installing the Atlas HCL extension, it automatically registers for all the languages that Atlas supports. These languages include:

| Name | Language ID | Description 
|----------|----------|----------|
| Atlas SQLite | atlas-schema-sqlite | SQLite dialect
| Atlas MySQL | atlas-schema-mysql | MySQL dialect
| Atlas PostgresQL | atlas-schema-postgresql | PostgresQL dialect
| Atlas MariaDB | atlas-schema-mariadb | MariaDB dialect
| Atlas Config File | atlas-config | Project Configuration

By default, the extension will auto-detect atlas.hcl files in your project.
However, if you have any schema files in your project, you must choose the language for them. You can do this in three ways in vscode: 

1. Click on "Change Language" for your schema files in this section at bottom. (See image below)

<img src="assets/installation-1.png">

2. Use the Command Palette and choose "Change language mode" (See image below)

<img src="assets/installation-2.png">

3. Use [Language Identifiers](https://code.visualstudio.com/docs/languages/identifiers)

## Wrapping Up
In this post, we've introduced **The Atlas HCL Extension**. There's much more inside, and we invite you to install it today and give it a try.

## What next?
* Follow the [Getting Started](https://atlasgo.io/cli/getting-started/setting-up) guide.
* Join our [Discord Server](https://discord.gg/zZ6sWVg6NT).
* Follow us [on Twitter](https://twitter.com/ariga_io).
