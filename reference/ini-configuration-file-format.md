# INI Configuration File Format

Reference for the INI file format as implemented by this package, aligned with PostgreSQL
`postgresql.conf` conventions.

## BNF Grammar

```bnf
<file>        ::= { <line> }
<line>        ::= <blank> | <comment> | <section> | <parameter> | <include>

<blank>       ::= { <whitespace> } <EOL>
<comment>     ::= { <whitespace> } ( "#" | ";" ) { <any-char> } <EOL>

<section>     ::= "[" <identifier> "]" { <whitespace> } [ <comment> ] <EOL>
<parameter>   ::= <key> { <whitespace> } [ <separator> ] { <whitespace> } <value>
                   { <whitespace> } [ <comment> ] <EOL>

<key>         ::= <identifier>
<separator>   ::= "=" | ":"
<value>       ::= <quoted-value> | <unquoted-value>
<quoted-value>    ::= "'" { <string-char> | "''" | "\'" } "'"
<unquoted-value>  ::= { <printable-char> }

<identifier>  ::= <letter> { <letter> | <digit> | "_" | "-" | "." | "$" }
<include>     ::= ( "include" | "include_if_exists" | "include_dir" ) <whitespace>
                   { <whitespace> } <value> <EOL>
```

## Comments

- `#` — standard line comment (PostgreSQL convention)
- `;` — alternative line comment (classic INI convention)
- Both may appear at the start of a line or inline after a value
- Inline `;` is ambiguous when the value itself contains semicolons

## Sections

- Declared with `[section_name]`
- Parameters before any section header belong to the **default (global) section**
- Section names are case-insensitive identifiers
- Empty sections (header with no parameters) are valid
- Duplicate sections are ignored (first one wins)
- PostgreSQL's [`postgresql.conf`](https://www.postgresql.org/docs/18/config-setting.html) uses a
  flat file (no sections), while its
  [connection service file](https://www.postgresql.org/docs/18/libpq-pgservice.html) does.

## Parameters (Key/Value Pairs)

- One parameter per line: `key = value`
- The `=` separator is canonical; whitespace around it is ignored
- Duplicate keys: last occurrence wins
- Keys are case-insensitive identifiers (`[A-Za-z_][A-Za-z0-9_.\-$]*`)

## Value Types

| Type      | Examples                                     | Notes                                      |
| --------- | -------------------------------------------- | ------------------------------------------ |
| Boolean   | `true` `false` `on` `off` `yes` `no` `1` `0` | Case-insensitive; unambiguous prefixes OK  |
| String    | `'syslog'` `plain_word`                      | Single-quote when value has spaces/special |
| Integer   | `100` `0xFF` `077`                           | Decimal, hex (`0x`), octal (`0`) prefixes  |
| Float     | `1.5` `0.001`                                | Standard decimal notation                  |
| With Unit | `128MB` `'120 ms'`                           | Memory: `B kB MB GB TB` (1024-based)       |
|           |                                              | Time: `us ms s min h d`                    |

## Quoting Rules

- **Unquoted** — sufficient for simple identifiers and numbers
- **Single-quoted** (`'...'`) — required when value contains spaces, `#`, `;`, or special chars
- Embedded single quotes: escape as `''` (double) or `\'` (backslash)
- Backslash escapes: `\\` for literal backslash inside quoted values
- Double quotes (`"..."`) may appear **inside** single-quoted strings for sub-quoting (e.g.,
  `search_path = '"$user", public'`)

## Blank Lines

Blank lines and lines containing only whitespace are ignored.

## Include Directives

- **`include = 'file'`** — includes a single file by exact path, error if file is unreadable
    - only non-directory files, any exentension is allowed, `.` hidden files are allowed
- **`include_if_exists = 'file'`** — like `include`, silently ignored if the file is unreadable
- **`include_dir = 'directory'`** — includes all `*.conf` files in the directory
    - only non-directory files ending in `.conf` that do not start with `.` are included
    - **lexicographical (sorted) order** by filename (e.g., `00base.conf` before `10overrides.conf`)

Glob patterns and wildcards (i.e. `*`, `**`) are **not supported**; paths must be exact files or
directories.

Relative paths are resolved relative to the directory containing the referencing config file.

Included files are processed inline — parameters override earlier values, and later includes
override earlier ones.

## Reserved / Special Words

- Boolean literals: `on`, `off`, `true`, `false`, `yes`, `no`, `1`, `0`
- Include directives: `include`, `include_if_exists`, `include_dir`
- Units (case-sensitive): `B`, `kB`, `MB`, `GB`, `TB`, `us`, `ms`, `s`, `min`, `h`, `d`
    - NOTE: the `inigo` implementation doesn't support this
