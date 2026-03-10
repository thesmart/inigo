# PGINI Reference (Agent-Optimized)

PGINI: PostgreSQL-compatible INI format. Mime: `text/x-pgini`. Encoding: UTF-8. Extension: `.conf`.

## Sections

- `[name]` header groups parameters until next header
- Names: case-insensitive identifiers
- Default section: parameters before any header (name = empty string, reopenable via `[default]`)
- Duplicate `[name]` reopens that section
- Empty sections valid

## Parameters

Format: `key = value` or `key : value` (one per line).

**Keys:** case-insensitive identifiers matching `[A-Za-z_][A-Za-z0-9_.\-]*`.

**Values:** whitespace-trimmed. Missing value = zero-value (`0`, `false`, or `""`). Duplicate keys:
last wins.

| Type    | Syntax                                                          |
| ------- | --------------------------------------------------------------- |
| Boolean | `true` `false` `on` `off` `yes` `no` `1` `0` (case-insensitive) |
| String  | `simple_word` or `'quoted string'`                              |
| Integer | `100` `0xFF` `077` (decimal, hex, octal)                        |
| Float   | `1.5` `0.001`                                                   |

**Quoting:** single-quotes required for values with spaces, `#`, `;`, or special chars. Escape `'`
as `\'` or `''`. Escape `\` as `\\`. Double-quotes allowed inside single-quoted strings (e.g.
`'"$user", public'`).

## Include Directives

- `include 'path'` — include file
- `include_if_exists 'path'` — include if exists, else skip
- `include_dir 'dir'` — include all `.conf` in dir (ascii order, skip dotfiles)

Relative paths resolve from the containing file's directory.

## Comments

`#` or `;` starts a line comment. Everything after is ignored until newline. Comment delimiters
inside quoted values are literal.

## Grammar (EBNF)

```ebnf
file           ::= line*
line           ::= blank | comment | section | parameter | include
blank          ::= WSP* EOL
comment        ::= WSP* [#;] any-char* EOL
section        ::= '[' identifier ']' WSP* comment? EOL
parameter      ::= key WSP* separator? WSP* value WSP* comment? EOL
include        ::= ('include' | 'include_if_exists' | 'include_dir') WSP+ quoted-path WSP* comment? EOL
key            ::= identifier
identifier     ::= letter (letter | digit | [_.\-])*
separator      ::= [=:]
value          ::= quoted-value | unquoted-value
quoted-value   ::= "'" (print-char | "''" | "\'")* "'"
unquoted-value ::= safe-char+
quoted-path    ::= "'" (abs-path | rel-path) "'"
abs-path       ::= '/' rel-path
rel-path       ::= path-component ('/' path-component)*
path-component ::= path-char+
path-char      ::= [^#x00-#x1F #x27 #x7F /]
letter         ::= [a-zA-Z]
digit          ::= [0-9]
print-char     ::= [^#x00-#x1F #x27 #x7F]
safe-char      ::= letter | digit | [_.\-]
WSP            ::= [#x20 #x09]
EOL            ::= #xD #xA | #xA | #xD
any-char       ::= [^#x00 #xA #xD]
```
