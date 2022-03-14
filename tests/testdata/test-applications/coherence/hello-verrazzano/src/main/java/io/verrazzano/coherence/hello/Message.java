// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package org.coherence.springboot.hello;

import java.io.Serializable;

import java.time.format.DateTimeFormatter;
import java.time.LocalDateTime;
import java.util.UUID;

public class Message implements Serializable {

    private String id;
    private String creationTime;
    private String message;

    public Message(String message) {
        this.id = UUID.randomUUID().toString().substring(0, 4);
        this.creationTime = LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss.SSS"));
        this.message = message;
    }

    public String getMessage() {
        return this.message;
    }

    public String getId() {
        return this.id;
    }

    public String getCreationTime() {
        return this.creationTime;
    }
}
