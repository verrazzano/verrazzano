// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package io.verrazzano.hello.resource;

import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

public class Greetings {

    @GET
    @Path("/greet")
    @Produces((MediaType.TEXT_PLAIN))
    public Response sayHello() {
	  return Response.status(200).entity("Hello Verrazzano").build();
    }
}
