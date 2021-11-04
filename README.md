# Alarm Digest

## Requirement discussion

This is how I implemented requirements defined in project description.

### Our broker cannot guarantee in-order delivery of the messages

Because of that whenever new message comes from `AlarmStatusChanged` it's `ChangedAt`
field is compared to the one stored ith previously received alarm for the same
(`UserID`, `AlarmID`) pair.

Because of this requirement more memory has to be used. If in-order delivery was guaranteed I
could remove alarm entry from memory whenever `CLEARED` status is received. In this case instead
I have to keep them to know if next possible message with different status is a real alarm or just
an outdated one.

Obviously messages received from different channels are not ordered either. So whenever `SendAlarmDigest`
request is received I send alarms immediately despite the fact that later I may receive `AlarmStatusChanged`
message from the past.

### Our broker can guarantee at-least-once delivery for all topics

Whenever I receive `AlarmStatusChanged` it is compared to the state stored previously so duplicates
don't cause any troubles - they are filtered immediately.

Whenever I send alarms to `AlarmDigest` they are marked as sent. But if I receive a duplicated message
from `SendAlarmDigest` new alarm set may be sent unintentionally if in the meantime some alarm was updated by
new `AlarmStatusChanged` message.

### Your solution should be capable of handling any amount of messages

The only accumulating part of the system is the database of current state of all the alarms.
This state may be sharded (see next part) between any number of servers. So it may serve infinite state
using infinite servers each providing finite memory. It will work as long as current state of
alarms triggered by single user may fit into memory of single server. I assumed this is always true.

### Your solution should be able to scale horizontally

Solution I provide uses sharding by `UserID`. It means state related to the user is kept on single machine
but different users may be targeted by different servers.

Only memory can be sharded here. CPU and network bandwidth can't. This limitation is caused
by the way you designed `AlarmStatusChanged` topic. Full discussion of this problem and possible solution
(how to redesign system) is in [doc/discussion.odt](doc/discussion.odt).

### It is very important that all active alarms are eventually sent to the user, alarms should not be lost

All the collected state is stored in RAM. Once application is terminated everything goes away.
Some solutions to fix it:
- Run many replicas of the same shard. It's possible with the current implementation but in such case duplicated
  messages are sent to `AlarmDigest`. Receiver should filter them out. Anyway, if there is a bug in the app
  all the replicas may panic at the same time.
- Use JetStream feature of NATS.
- Alarm producers could resend active alarms from time to time to rebuild the state if some servers were down.

### Ideally a user shouldn’t receive the same alarm twice, if its status has not changed since the last digest email.

Each time alarms are send they are marked as sent. So next time alarm is sent only if update with different status
was received.

### Active alarms should be ordered chronologically (oldest to newest)

I sort them before sending

## Architecture

### Global sharding

Data are sharded across servers by `UserID` field. All the requests from `AlarmStatusChanged` and `SendAlarmDigest`
related to the same user are processed by the same node. This node publishes results to `AlarmDigest`.

By doing this state of any size may be stored in the system by distributing it between many servers, as long as state of
single user fits into one server, which should never be a problem.

Because in producer-subscriber model applied here all messages are received by all servers each server silently discards
messages which should be handled by different shards. This is suboptimal this topic is discussed in details in [doc/discussion.odt](doc/discussion.odt).

### Local sharding

On a node data are sharded across many goroutines. Whenever message comes to the node it is delivered to appropriate
goroutine. All the requests from `AlarmStatusChanged` and `SendAlarmDigest`
related to the same user are processed by the same goroutine. This goroutine publishes results to `AlarmDigest`.
Default number of local shards is equal to the number of logical CPUs on the machine.

By doing this concurrency of processing is increased. Local state is distributed across goroutines so those parts may be
handled concurrently. At the same time each local shard is accessed by single goroutine only so there is no need
to maintain mutexes during data access.

### Dispatching messages

To deliver message to correct shard (global or local) `UserID` is converted to `uint64` number which is then
divided modulo by the number of shards. The result is a number in range `[0, NumOfShards)`.
In global sharding only messages with matching shard ID are processed. In local sharding message is delivered
to appropriate goroutine responsible for particular local shard.

### User state

Each global and local shard manages state related to matching users. For each user, list of alarms is stored.
Each alarm holds the last state of the alarm which is updated each time message from `AlarmStatusChanged` comes.

These rules apply when update is applied:
- only if time in update is greater than the one already stored, alarm is updated - it is ignored otherwie
- if new status is `CLEARED` flag `ToSend` is reseted
- if old and new status are equal, `ToSend` flag is not set (old value is kept)
- otherwise `ToSend` is set to `true`

### Sending state

Whenever request comes from `SendAlarmDigest` alarms with `ToSend` flag set to `true` are groupped, sorted and sent.

### Unit tests

All logic I considered important is unit-tested. I didn't write tests for negative scenarios when for ex. data in incorrect
format may come, despite the fact that this logic is implemented. I just don't want to invest more time in obvious stuff.

The only important missing part is testing `natsConnection` type, though only very minimal set of instructions is placed
there. Writing tests for it would require implementing a mock of NATS client. I hope all the other code I present proves
that I could do that. i didn't only because I've already invested significant amount of time in this project.

## Libraries

### Build

In [build] directory I use my build system I developed for fun some time ago. You may find it [here](https://github.com/wojciech-malota-wojcik/build)

### Inversion of control

It is not popular in go apps to use IoC containers but I do it from time to time. The implementation I use in this project is available [here](https://github.com/wojciech-malota-wojcik/ioc)

### Parallelism 

For managing goroutines I use best-ever library for the job (I'm co-author) available [here](https://github.com/ridge/parallel)

## Usage

To run the app:

```
go run ./cmd
```

Available options:

- `--help` - prints help message
- `--verbose`, `-v` - turns on verbose logging
- `--nats-addr` - address of NATS server, may be specified many times to provide access to more nodes forming cluter
- `--shards` - total number of global shards
- `--shard-id` - number representing global shard handled by this instance
- `--local-shards` - number of local shards (processing goroutines) to start

All parameters have reasonable default values for running system with single global shard.