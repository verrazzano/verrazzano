// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// File containing groovy functions to be used by Jenkinsfile in this directory

// import java.text.SimpleDateFormat
// def dateFormat = new SimpleDateFormat("MM/dd/yyyy HH:mm:ss")

def testUtilsFile() {
    return "success"
}

def log(status, msg) {
    // timestamp = dateFormat.format(new Date())
    println("${status} - ${msg}")
}

return this
