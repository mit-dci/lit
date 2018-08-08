# Lit Integration Tests

Lit's integration test suite was quite outdated, so as of 2018-08-03 we started
rewriting them.  Now they're waaay cooler and easier to write.

All of the tests are in Python and built on the the `testlib` library, which
manages creating instances of lit and and bitcoind and getting them to talk to
each other without worrying about the details.

The actual tests are in each of the `itest_foo.py` files.

## Deps

You'll need the `requests` library, and `bitcoind` on your PATH, but that's
about it.

```sh
pip3 install requests
```

## Running the tests

There's a separate shell script that can manage executing all the tests.  It
also manages setting up data directories for tests, which are stored in `_data`.

```sh
./runtests.sh <tests...>
```

You can specify specific tests or pick which ones you want to run out of the
list in `tests.txt`.  It also supports Bash's job control, so if you C-c out of
the script it'll properly handle cleaning up the current test and skipping over
the remaining ones.

## Envvars

* `LIT_OUTPUT_SHOW` - Tell the testlib to show output from all Lit(s)

* `LIT_ID_SHOW` - Used with the above, *only* show output from this Lit

* `LIT_ITEST_ROOT` - Data dir path, only works when running tests directly.

## Notes

Some of the tests have a `_reverse` suffix.  This means that they do mostly the
same thing as the test without the `_reverse` suffix, but the difference between
them is the opposite node closes the channel between the two tests.  For these
the actual test code is probably in `test_combinators.py` and that's used as a
library in the main test script.  You can see the settings in the main script as
as pass slightly different arguments to the function that runs it.
