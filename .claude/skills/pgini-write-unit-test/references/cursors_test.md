# cursors_test.go

Tests for `pgini/cursors.go` — covers `RootCursor` and `FileCursor` types.

## Specification

When writing/editing/fixing tests, consider first what the [specification](../pgini-agents.md)
intends for the implementation to do. The test should verify that implementation adheres to the
specification by comparing an expected result (specification) with actual result (implementation).

If the two differ, it is probably due to a bug in the code.

## Tests to Consider

1. Verify that the iterator can go from the start to the very end of a INI file.
1. Test that adding the same include path will result in an error (duplicate detection)
