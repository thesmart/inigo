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

**Keys:** case-insensitive identifiers, see [PGINI Grammar](#pgini-grammar) for details.

**Values:** whitespace-trimmed. Missing value = zero-value (`0`, `false`, or `""`). Duplicate keys:
last wins.

| Type    | Syntax                                                          |
| ------- | --------------------------------------------------------------- |
| Boolean | `true` `false` `on` `off` `yes` `no` `1` `0` (case-insensitive) |
| String  | `simple_word` or `'quoted string'`                              |
| Integer | `100` `0xFF` `+8kB` `-1` (decimal, hex; optional sign & unit)   |
| Float   | `1.5` `0.001`                                                   |

**Unquoted values:** Simple values containing latin alpha-numeric chars and `[-._:/+]` are not
required to be enclosed in quotes.

**Quoted values:** Enclose values in single-quotes to all for UTF-8 characters.

- single-quote `'`, i.e. `\'` or `''`
- backslash `\` as `\\`
- control characters use C-style backslash escapes: `\b` (backspace), `\f` (form feed), `\n`
  (newline), `\r` (carriage return), `\t` (tab); all other control characters (U+0000–U+001F,
  U+007F) use octal encoding `\OOO` (1–3 octal digits)

## Include Directives

- `include 'filepath'` — include file
- `include_if_exists 'filepath'` — include if exists, else skip
- `include_dir 'dirpath'` — include all `.conf` in dir (ascii order, skip dotfiles)

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
identifier     ::= letter ( letter | digit )*
separator      ::= [=:]
value          ::= quoted-value | unquoted-value
quoted-value   ::= "'" (print-char | escape-seq)* "'"
escape-seq     ::= "\\" | "\'" | "''" | "\b" | "\f" | "\n" | "\r" | "\t" | octal-escape
octal-escape   ::= "\" octal-digit octal-digit? octal-digit?
octal-digit    ::= [0-7]
unquoted-value ::= safe-char+
quoted-path    ::= "'" (abs-path | rel-path) "'"
pathname       ::= abs-path | rel-path
abs-path       ::= '/' rel-path?
rel-path       ::= path-segment ( '/' path-segment )*
path-segment   ::= segment-char+
segment-char   ::= [^#x00-#x1F #x27 #x7F /]
letter         ::= [a-zA-Z_]
digit          ::= [0-9]
print-char     ::= [^#x00-#x1F #x27 #x5C #x7F]
safe-char      ::= letter | digit | [_.\-:/+]
WSP            ::= [#x20 #x09]
EOL            ::= #xD #xA | #xA | #xD
any-char       ::= [^#x00 #xA #xD]
```
