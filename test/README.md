# Lit integration tests

This directory contains lit integration tests, which start lit and bitcoind/litecoind nodes and test functionality.

### Directory Structure

- `lit_test_framework.py` is the test framework. Individual test cases should import this and subclass the LitTest class
- `litnode.py` contains a class representing a lit node. This can be used to start/stop the node and communicate with it over the websocket RPC.
- `bcnode.py` contains a class representing a bitcoind node or litecoin node. These can be used to start/stop the bitcoin/litecoin nodes and communicate with them over RPC.
- `test_*.py` individual tests cases.

### Dependencies

- websocket-client (`pip install websocket-client`)
- requests (`pip install requests`)
- bitcoind and litecoind on the path

### Running tests

Run tests with `./test_[testname].py`

### Adding tests

New tests should be named `tests_[description].py`. They should import the `LitTest` class from `lit_test_framework.py`. The test should subclass the `LitTest` class and override the `run_test()` method to include its own test logic. See `test_basic.py` for an example.
