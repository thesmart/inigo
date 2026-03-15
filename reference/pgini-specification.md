# PGINI: A Specification for PostgreSQL-Compatible INI Configuration

What is an INI configuration file? You've probably seen one:

```ini
# example ini configuration
name = myapp
host = localhost
port = 1337
```

Simple. Readable. Human-friendly. The INI format has been a staple of software configuration for
decades: from early Windows system files to web application configs. At its core, it's just keys,
values, and the occasional section header. Almost anyone can read one. Almost anyone can write one.
And that's precisely the problem.

**There is no standard.** No RFC. No ISO document. No ECMA specification. Nothing that says what an
INI file _actually is_. Every application, every library, every language decides for itself. Can
values be quoted? Are keys case-sensitive? Do you use `=` or `:`? Are comments `#` or `;`? What
about multiline string? Nested sections? Types?

Ask five parsers and you'll get six opinions. So you might say, we haven't INI standard. (Okay,
sorry for that.)

Can we do better than convention and vibes?

## Settling on one: PostgreSQL

For over two decades, PostgreSQL has shipped with
[pg_service.conf](postgresql.org/docs/18/libpq-pgservice.html). It's not written up as a formal
specification, but it is popular and battle-tested. It has clear and simple rule about quoting,
types, includes, and comments.

## PGINI Configuration Standard

PGINI (pronounced: "pee-gee-nee"):

- mime-type: `text/x-pgini`
- encoding: `UTF-8`
- preferred file extensions: `.conf` or `.pgini`
    - not required: may have any file extension

PGINI files contain:

- [Parameters](#parameters): key-value pair definitions
- [Sections](#sections): optional named groups of parameters
- [Includes](#include-directives): link configuration files into one
- [Comments](#comments): for documentation

There is a full [PGINI BNF Grammar](#pgini-grammar).

### Sections

A section is a group of parameters.

- Sections are declared with a `[name]` section header
- Section names are case-insensitive identifiers
- Default group is named empty-string, but can implicitly used via `[default]`:
    - Contains parameters defined prior to any section header
    - Files always contain at least one section (i.e. default)
- Parameters following a section header belong to that section until the next section header
- Empty sections (header with no parameters) are valid
- Duplicate sections re-open prior sections

Example:

```ini
# These belong to the default section
host = localhost
port = 5432

# The "mydb" section
[mydb]
dbname = app

# Re-opens default section
[default]
dbname = foobar
```

### Parameters

A parameter is a `key = value` pair within a section.

- One parameter per line: `key = value`
- `=` - key/value separator (PostgreSQL convention), whitespace around it is ignored
- `:` - alternative key/value separator, whitespace around it is ignored
- Duplicate parameter: last occurrence wins
- Keys are case-insensitive identifiers, see [PGINI Grammar](#pgini-grammar) for details
- Missing value (key with no separator or value) is the zero-value for the type: `0`, `false`, or
  empty-string
- Leading and trailing whitespace around values is ignored (values are trimmed)

Examples:

```ini
# = separator (canonical)
host = localhost
# : separator (alternative)
host: localhost

# whitespace around separator is ignored
host=localhost
host =localhost
host= localhost
host : localhost

# single-quoted values (required for special chars)
greeting = 'hello world'
message = 'foobar\'s bingbat'
# backslash escape
path = 'C:\\data'
# last key occurrence wins
port = 1111
port = 2222
```

#### Parameter Values

**Type** is not inferred, but defined by the host application that ingests the INI file. This chart
explains how datatypes are expressed in the INI:

| Type    | Examples                                     | Notes                                          |
| ------- | -------------------------------------------- | ---------------------------------------------- |
| Boolean | `true` `false` `on` `off` `yes` `no` `1` `0` | Case-insensitive; unambiguous prefixes OK      |
| String  | `'quoted string'` `simple_word`              | Use single-quotes when value has special chars |
| Integer | `100` `0xFF` `077`                           | Decimal, hex (`0x`), octal (`0`) prefixes      |
| Float   | `1.5` `0.001`                                | Standard decimal notation                      |

**Unquoted values:** Simple values containing latin alpha-numeric chars and `[-._:/]` are not
required to be enclosed in quotes.

**Quoted values:** Enclose values in single-quotes (e.g. `'value'`) allows for the complete UTF-8
character set. Here are the rules for escaping:

- single-quote `'`, i.e. `\'` or `''`
- backslash, i.e. `\` as `\\`
- control characters use C-style backslash escapes: `\b` (backspace), `\f` (form feed), `\n`
  (newline), `\r` (carriage return), `\t` (tab); all other control characters (U+0001–U+001F,
  U+007F) use octal encoding`\OOO` (1–3 octal digits)

### Include Directives

- `include 'filepath'` - include another INI file
- `include_if_exists 'filepath'` - include if exists, else skip
- `include_dir 'dirpath'` - include all `.conf` in dir (ascii order, skip dotfiles)

Included files are processed as if inserted at the line of the include directive.

Relative paths resolve from the containing file's directory.

**Breaking Change:** PostgreSQL allows for unquoted paths; we require single-quoted paths.

### Comments

A single-line string of documentation. The parser will simply ignore any data after a comment
delimiter. Comment delimiters cannot exist within quoted text regions.

- `#` - standard line comment (PostgreSQL convention)
- `;` - alternative line comment (classic INI convention)
- Anything after a comment delimiter is ignored until newline (`\n`) is reached

## Breakng Changes

PGINI implementation differs from
[PostgreSQL implementation](https://raw.githubusercontent.com/postgres/postgres/refs/heads/master/src/backend/utils/misc/guc-file.l)
in the following ways:

1. Identifiers: same as PG, except ASCII-only
2. Include directives: use quoted paths (e.g. `include '<PATH>'`) vs PG optionally quoted paths
3. PGINI allows for `#` and `;` as comment delimiter, PG uses `#` only
4. PGINI allows for `=` and `:` as parameter identifier/value delimiter, PG uses `=` only
5. PGINI uses these boolean values:
    - `true`: "t", "1", "true", "on", "y", "yes"
    - `false`: "f", "0", "false", "off", "n", "no"
6. PGINI doesn't have built-in support for expontent numbers or types (e.g `kb`, `MB`)

## PGINI Grammar

This grammar formally defines what sequences of characters are valid:

```ebnf
file           ::= line*
line           ::= blank | comment | section | parameter | include

blank          ::= WSP* EOL
comment        ::= WSP* [#;] any-char* EOL

section        ::= '[' identifier ']' WSP* comment? EOL
parameter      ::= key WSP* separator? WSP* value WSP* comment? EOL
include        ::= ( 'include'
                   | 'include_if_exists'
                   | 'include_dir' ) WSP+ quoted-path WSP* comment? EOL

key            ::= identifier
identifier     ::= letter ( letter | digit )*
separator      ::= [=:]
value          ::= quoted-value | unquoted-value

quoted-value   ::= "'" (print-char | escape-seq)* "'"
escape-seq     ::= "\\" | "\'" | "''" | "\b" | "\f" | "\n" | "\r" | "\t" | octal-escape
octal-escape   ::= "\" octal-digit octal-digit? octal-digit?
octal-digit    ::= [0-7]

unquoted-value ::= safe-char+

quoted-path    ::= "'" ( abs-path | rel-path ) "'"
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
