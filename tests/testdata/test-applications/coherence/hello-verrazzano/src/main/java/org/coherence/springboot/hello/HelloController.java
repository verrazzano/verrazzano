// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package org.coherence.springboot.hello;

import com.oracle.coherence.spring.configuration.annotation.CoherenceMap;

import com.tangosol.net.Coherence;
import com.tangosol.net.NamedMap;

import org.springframework.beans.factory.annotation.Autowired;

import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class HelloController {

    @Autowired
    private Coherence coherence;

    @CoherenceMap
    private NamedMap<String, Message> messages;

    @GetMapping("/hello")
    public String greet() {
        if (!messages.isEmpty()) {
            StringBuilder sb = new StringBuilder();
            messages.entrySet().stream().forEach((entry) -> {
                String messageId = entry.getKey();
                Message oneMessage = entry.getValue();
                sb.append(oneMessage.getCreationTime() + " - " + oneMessage.getMessage()).append(System.getProperty("line.separator"));
            });
            return sb.toString();
        }
        return "Hello Verrazzano";
    }

    @RequestMapping(value = "/postMessage", method = RequestMethod.POST)
    public ResponseEntity<String> postMessage(@RequestBody String message) {
        Message newMessage = new Message(message);
        messages.put(newMessage.getId(), newMessage);
        return ResponseEntity.status(HttpStatus.CREATED).build();
    }
}
