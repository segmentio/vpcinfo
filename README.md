# vpcinfo
Experimental model for distributing non-sensitive VPC information.

## Motivation

Having access to network topology information is critical to large network
infrastructures. However, the solutions for accessing this information are
often limited in one way or another. At Segment, we explored multiple approaches
to solving this problem, each came with its set of stengths and weaknesses.
Overall, there are two ways in which network topology information can be
collected and leveraged: within processes maintaining open connections, and
out-of-band by some monitoring system.

### In-process

Consul has been a popular choice to support advanced network routing models at
Segment. Services register and add extra metadata (for example, to describe the
availability zone they are part of), and clients use consul to discover targets
and apply smart routing like zone-affinity on the traffic.

This model works well and has the advantage to be done on the application side,
rich information about the netework routes can be injected in telemetry.

The downside of using a third-party system like Consul is that it is hard to
cover resources that it is not deployed on. In today's Cloud infrastructures,
we have heterogenous systems, sometimes deployed as managed services, functions,
containers, etc... We found it very hard to use systems like Consul to cover all
these use cases.

Accessing Cloud provider APIs could be seen as an alternative solution to
collect insight into the environment that a program is running in.
Unfortunately these APIs are often not designed for heavy read access,
and apply low request-rate limits. These limits also apply across the board,
so one abusive actor can end up impacting other use cases. In our experience,
we have been more successful avoiding to access to Cloud APIs in production
systems, reserving them for infrastructure configuration only.

### Out-of-band

Many solutions exist that provide network topology information by monitoring
network activity (using using techonologies like ebpf). We experimented with a
couple of third-party solutions and built some prototypes of our own as well.

The strength of this approach is it applies across an entire infrastructure
without having to make code changes to the services being monitored, we were
able to quickly get deep insight into some of our networks.

The weakness of this approach is in how much data can be read by the monitoring
tool, usually limited to _raw_ data. In our experience, most of the value is in
connecting network topology to high-level information that only the applications
have access to, because we can then tie product use cases to resource usage.

## Design

The solution in this package relies on two components: a Terraform module and a
Go package. The focus of this package is only on providing programs access with
information about the network they are running in.

### Terraform

The repository provides a terraform module that plugs into a cloud VPC and
injects DNS TXT records that expose information about the network topology.
Data is stored URL-encoded in the TXT records, each record representing one
VPC resource.

Here is an example of how to deploy the resources to a VPC on AWS:

```hcl
module "vpcinfo" {
  source = "git@github.com:segmentio/vpcinfo//aws"
  vpc_id = "${aws_vpc.main.id}"
}
```

### Client

The second component is a Go package which supports querying the TXT records and
extracing information from them, adding a smaller caching layer for efficiency
since network topology does not change very often. We provide a Go client
because this is what most of our infra is built in, but clients can be written
in any languages and are pretty simple to construct: DNS client libraries and
URL decoders are plenty out there.

Here is an example of looking up the list of subnets in the VPC:

```go
package main

import (
    "fmt"

    "github.com/segmentio/vpcinfo"
)

func main() {
    subnets, _ := vpcinfo.LookupSubnets()

    for _, s := range subnets {
        fmt.Println(s)
    }
}
```

### DNS

We chose to expose the VPC information over DNS because it is a protocol that
operating systems have lots of infrastructure for, and DNS clients are usually
available in every major programming language, making it simple to build
programs that leverage this information.

By default, the Terraform module configures a new DNS zone called `vpc.local`
where DNS records are written. The Go client package is also configured to use
this default domain name. Relying on a well-known domain, independent from
corporation-specific names, helps deploy this solution in various environments
without having to take on extra configuration options in every service that
needs access to VPC information.
