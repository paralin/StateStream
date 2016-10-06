State Streams
=============

A state stream is a time series log of entries consisting of two types of entries:

 - Snapshot - a full snapshot of a state object
 - Mutation - a change / mutation on that state object

This package provides the following functionality:

 - [x] Accept a snapshot and append a change or keyframe to the stream.
 - [x] Calculate state at any given timestamp
 - [x] Enforce a maximum granularity (batch changes together).
 - [ ] Re-structure periods of time:
   - Change keyframe frequency
   - Change maximum granularity (change distribution)

This package has a general interface for interacting with a stream, to allow for pluggable storage backends.

Cursors
=======

In a state stream, a cursor is a calculated state tree at a particular point in time. It maintains context of the calculations used to derive the state at that point in time, such that the cursor can be moved forward or backward in time with minimal re-calculation or database churn.

Internally the State Stream system uses a cursor pointing to the point in time immediately following the latest known state change. Then, this cursor can be used to write to the stream in a fast way.

Streams can be initialized in a few different modes:

 - "write" mode: track the latest state, perform writes
 - "read-forward" mode: calculate state at a given time, don't store data to rollback the stream in time.
 - "read-bidirectional" mode: calculate state at a given time, and remember how to reverse changes such that rolling the stream backward in time is fast.

Write Cursors
=============

A write cursor:

 - Always sits at the "now" timestamp
 - Stores the last snapshot + current mutation (last one in list)
