Playwright cleaner
==================

This is a Go utility to reduce the size of Playwright traces, by removing files
you don't need. Simply pass it the name of a playwright-report directory, and it
will by default take out:

- resource files larger than 1 MB
- function args larger than 1 MB from the trace file

To use this, simply `go run cmd/playwright-cleaner <path to playwright-report>`
or use one of the binary releases.

Or, to use this in Github Actions after running the Playwright tests and
before uploading the artifacts:

```
    - name: Run Playwright tests
      run: yarn run test

    - name: Prune Playwright test report
      if: always()
      run: |
        wget https://github.com/sgielen/playwright-cleaner/releases/download/v1.0.0/playwright-cleaner_Linux_x86_64.tar.gz
        tar -xzf playwright-cleaner_Linux_x86_64.tar.gz
        ./playwright-cleaner ./playwright-report

    - uses: actions/upload-artifact@v3
      if: always()
      with:
        name: playwright-report
        path: ./playwright-report/
        retention-days: 14
```

See also: https://github.com/microsoft/playwright/issues/20157
