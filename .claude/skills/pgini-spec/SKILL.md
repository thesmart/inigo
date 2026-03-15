# pgini-spec

Activate when implementing or modifying PGINI parsing, marshaling, lexing, or any code that must
conform to the PGINI file format specification. Triggers: pgini, ini parsing, ini format, conf file,
GUC, postgresql.conf, quoting, escaping, lexer. Do NOT activate for general Go questions unrelated
to the INI format.

## Instructions

Read the following files to load full context for PGINI work:

1. Project specification: `README.md`
2. PGINI format specification (agent-optimized): `reference/pgini-agents.md`
3. Development setup and SDLC: `CONTRIBUTING.md`
4. PG lexer — token definitions and scanner rules (lines 71–107):
   `reference/pg-backend-guc-file.l:71:107`
5. PG lexer — `ParseConfigFp` conf file parser (lines 318–568):
   `reference/pg-backend-guc-file.l:318:568`
6. PG lexer — `DeescapeQuotedString` de-quoting function (lines 649–742):
   `reference/pg-backend-guc-file.l:649:742`

After reading, use these references as the authoritative source for how PGINI files are parsed,
tokenized, quoted, and escaped. Any implementation must conform to the behaviors described in these
references.
