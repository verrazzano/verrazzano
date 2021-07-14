# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
curl -u verrazzano:UftokHmESBAi8bDRCtqA "https://prometheus.dev.verrazzano.sauron.us-ashburn-1.oracledx.com/api/v1/query?query=prometheus_build_info" > metrics01.json
sed "s/'METRICS_01'/$(cat metrics01.json)/" tests/reports/templ.html > tests/reports/TestResult.html
