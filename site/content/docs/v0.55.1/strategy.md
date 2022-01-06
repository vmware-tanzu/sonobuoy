# Sonobuoy Strategy

## The problem
Users of Kubernetes value its ability to automate running complex systems at scale. However, little Kubernetes cluster validation is automated. As a result, skilled Kubernetes operators spend needless time performing manual tasks. Just as automated unit tests frontload the work of testing software by having the developer codify how to check that their code is functional, Kubernetes would benefit from automated test suites that allow skilled Kubernetes operators and developers to codify once how to test cluster capabilities and then have those checks run by whichever Kubernetes user needs this.

Examples:
 - Kubernetes experts still perform many manual tasks:
 - A security auditor works with a Kubernetes expert to determine whether a Kubernetes cluster is CIS compliant.
 - After a Kubernetes cluster is restored, the backup admin spends time running manual tests to confirm the cluster is functional.
 - A support engineer manually checks different areas of functionality to diagnose an Kubernetes cluster issue while on a Zoom call with a customer.
 - To install developer tooling, for example a cloud native runtime like Knative, a Kubernetes expert from the platform team is needed to validate that the user has the required installation privileges.

Worse, these steps may be skipped, leading to wasted time, technical debt, or even risk of operational failure:

 - The CIS audit is skipped, leaving security vulnerabilties in the Kubernetes cluster.
 - The backup admin doesn’t check the restored cluster and only realizes later through user-reported bugs that the container registry didn’t correctly re-attach to its S3 image store.
 - The support engineer tries reading logs to diagnose the problem, but because of the complexity of the issue, the logs lead in the wrong direction.
 - The platform operator installing the developer tooling (Cloud Native Runtime) doesn’t know how to check if the Kuberentes cluster has a load balancer, so they either spend significant extra time researching how to check this or skip the step, but then encounter myriad problems later when the tooling doesn’t install correctly.

## Who would benefit from solving this problem?
 - Users of these automated test suites reduce risk to the business, since alternatives are to use less reliable manual testing or skipping such testing altogether. Users also potentially save time, and thus money, that would have been spent in manual testing.
 - Skilled Kubernetes operators can spend their time on innovative work that only they can do, rather than repetitive manual checks. This, of course can translate into increased revenue or decreased costs for their organization through prioritization of experts’ time.
 - Organizations can more reliably get started with Kubernetes, given the existence of automated validation checks alongside their deployment and operation processes.  In this way, they can slowly grow their platform teams to have Kuberentes knowledge, rather than having to make a big up front investment in operators skilled in Kubernetes.

## The solution
A tool exists that solves this exact problem. Sonobuoy, well-known in the Kubernetes community as the CNCF-recommended way to run the Kubernetes conformance tests, has an underused pluggable architecture that allows users to build automated test suites and run them in Kubernetes clusters. Teams can use the Sonobuoy plugin skeleton to easily and quickly create customized test suites that suit their and other users’ needs.

## Call to Action

Please help us increase the number of Sonobuoy use cases!
The Sonobuoy team has started pairing with teams, to help them develop the suites they need.
While teams can indeed create the suites themselves, we want to understand users of the plugin skeleton so we can better encourage adoption.
We are learning about each teams' use cases and how to make it as easy as possible to create and run Sonobuoy customized test suites.
Once we have several functional, often-used test suites, we will explore ways to organically grow the number of teams and Kubernetes users using this Sonobuoy feature, such as promoting this Sonobuoy feature to the larger Kubernetes community. 

Be in touch if you are using Sonobuoy beyond conformance testing - we want to learn more. And we will do our best to help you if you are brainstorming a new, creative application of Sonobuoy's automated cluster validation.