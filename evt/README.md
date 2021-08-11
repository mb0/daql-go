evt
===

Package evt provides servers and tools for event sourcing. Event sourcing in this context means
a data model that is based on a sequence of events that can recreate the data model at any point
int time.

An `Event` consists of a string topic, a key, command a revision time and optionally an argument.
Daql uses mostly dumb events where the topic is a model name, the key a primary id and command a
generic create, update or delete command. It can be used for more specific events, those however
must resolve to a sequence of generic events, to allow a clean interface for backends.

`Ledger` represents a sequence of events ordered by revision. `Publisher` is a ledger that publishes
transactions and assigns new revisions to events. `Replicator` is a replicated ledger and the
`LocalPublisher` is a `Replicator` that can publish some events locally.

The event and ledger revision is a timestamp with millisecond granularity. It is usually the arrival
time of the event but must be greater than the last revision in the persisted ledger.

Every transaction generates an audit log entry that has extra information. Backup and restore
require both audit and event logs, as well as other data not covered by the event sourcing.

`Server` provides hub services to subscribe and publish to a ledger. Servers usually uses a ledger
implementation that updates the latest model state to support queries without event aggregation for
most operations. We might at some point introduce stateless topics, that have their only persistent
representation in the ledger.

`Satellite` connects to a server hub, replicates events, and manages local subscriptions.
Satellites can publish authoritative events locally to support offline use to some extent.
