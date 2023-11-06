### T-SQL parser based on ANTLR4

#### Resources

1. T-SQL syntax: https://learn.microsoft.com/en-us/sql/t-sql/language-elements/language-elements-transact-sql
2. Grammar file: https://github.com/antlr/grammars-v4/tree/master/sql/tsql

#### Run codegen

1. Install `antlr4`: https://github.com/antlr/antlr4/blob/master/doc/getting-started.md#unix
2. Run:
```bash
antlr4 -Dlanguage=Go -package tsqlparse -visitor TSqlLexer.g4 TSqlParser.g4 \
  && rm *.interp *.tokens
```