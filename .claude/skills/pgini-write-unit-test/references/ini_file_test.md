# ini_file_test.go

Tests for `pgini/ini_file.go` — covers `IniFile`, `Section`, and `Param` types.

## Specification

When writing/editing/fixing tests, consider first what the [specification](../pgini-agents.md)
intends for the implementation to do. The test should verify that implementation adheres to the
specification by comparing an expected result (specification) with actual result (implementation).

If the two differ, it is probably due to a bug in the code.
