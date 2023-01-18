Playwright cleaner
==================

This is a Go utility to reduce the size of Playwright traces, by removing files
you don't need. Simply pass it the name of a playwright-report directory, and it
will by default take out:

- resource files larger than 1 MB
- function args larger than 1 MB from the trace file

See also: https://github.com/microsoft/playwright/issues/20157
