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

This specification defines a simple super-set of features on top of PostgreSQL's conventions.

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
- Keys are case-insensitive identifiers (`[A-Za-z_][A-Za-z0-9_.\-]*`)
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

**String quoting rules:**

- Unquoted: sufficient for simple values
- Single-quoted (`'...'`): required when value contains spaces, `#`, `;`, or special chars
- Escaping single quotes: `\'` (backslash) or `''` (double)
- Backslash escapes: `\\` for literal backslash inside quoted values
- Double quotes (`"..."`) may appear **inside** single-quoted strings for sub-quoting (e.g.,
  `search_path = '"$user", public'`)

### Include Directives

- `include 'filename'` - include another INI file
- `include_if_exists 'filename'` - include another INI file if it exists, otherwise ignore
- `include_dir 'directory'` - include all `.conf` and `.pgini` files in directory processed in ascii
  order, files starting with `.` are excluded

Included files are processed as if inserted into the configuration file at that point.

Relative paths are resolved relative to the directory of the file containing the directive.

### Comments

A single-line string of documentation. The parser will simply ignore any data after a comment
delimiter. Comment delimiters cannot exist within quoted text regions.

- `#` - standard line comment (PostgreSQL convention)
- `;` - alternative line comment (classic INI convention)
- Anything after a comment delimiter is ignored until newline (`\n`) is reached

### Reserved / Special Words

- Boolean literals: `on`, `off`, `true`, `false`, `yes`, `no`, `1`, `0`
- Include directives: `include`, `include_if_exists`, `include_dir`

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
identifier     ::= letter ( letter | digit | [_.\-] )*
separator      ::= [=:]
value          ::= quoted-value | unquoted-value

quoted-value   ::= "'" ( print-char | "''" | "\'" )* "'"
unquoted-value ::= safe-char+

quoted-path    ::= "'" ( abs-path | rel-path ) "'"
pathname       ::= abs-path | rel-path
abs-path       ::= '/' rel-path?
rel-path       ::= path-segment ( '/' path-segment )*
path-segment   ::= segment-char+
segment-char   ::= [^#x00-#x1F #x27 #x7F /]

letter         ::= [a-zA-Z]
digit          ::= [0-9]
print-char     ::= [^#x00-#x1F #x27 #x7F]
safe-char      ::= letter | digit | [_.\-]

WSP            ::= [#x20 #x09]
EOL            ::= #xD #xA | #xA | #xD
any-char       ::= [^#x00 #xA #xD]
```
