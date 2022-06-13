In order to run these make targets locally, copy env-template.sh someplace and customize it to your liking.
The template file describes what minimum variables are needed to be able to run the targets.

To create a single KIND cluster and install a platform operator:

$ make setup

To install verrazzano run

$ make install

To run a test, you can run any test suite using the TEST_SUITES variable and the "test" target:

$ TEST_SUITES="verify-install/..." make test 

There are some convenience targets for some common acceptance tests, e.g.:

$ make verify-install
$ make verify-infra
$ make verify-infra-all

See acceptance-tests.mk for a full list.

