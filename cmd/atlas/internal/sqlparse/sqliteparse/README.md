### SQLite parser based on ANTLR4

#### Resources

1. SQLite syntax: https://www.sqlite.org/syntaxdiagrams.html
2. Grammar file: https://github.com/antlr/grammars-v4/tree/master/sql/sqlite

#### Run codegen

1. Install `antlr4`: https://github.com/antlr/antlr4/blob/master/doc/getting-started.md#unix
2. Run:
```bash
antlr4 -Dlanguage=Go -package sqliteparse -visitor Lexer.g4 Parser.g4 \
  && mv _lexer.go lexer.go \
  && mv _parser.go parser.go \
  && rm *.interp *.tokens
```