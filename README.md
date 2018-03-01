# Network Error Logging collector

This repository implements a collector for the [Reporting][] and [Network Error
Logging][] (NEL) specifications.  These specs allow site owners to instruct
browsers and other user agents to collect and report on reliability information
about the site.  This gives you the same information as you'd get from your
server logs, but collected from your clients.  This client-side data set will
include information about failed requests that never made it to your serving
infrastructure.

[Reporting]: https://wicg.github.io/reporting/
[Network Error Logging]: https://wicg.github.io/network-error-logging/

This repository provides a full working implementation of the collector side of
the specs.  If you run this collector behind a publicly available URL, you can
use that URL in NEL configuration headers for your web site or service.
NEL-compliant user agents will send reports about requests to your domain to
this collector.  You can then route them to a metrics or logs collection service
for further analysis.
