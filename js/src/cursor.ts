import { IStorageBackend } from './backend';
import {
  StreamEntry,
  StreamEntryType,
  Clone,
  DeepEqual,
  StateData,
} from './entry';
import {
  IRateConfig,
} from './config';
import {
  Subject,
} from 'rxjs';
import {
  NoDataError,
} from './errors';
import {
  applyMutationObject,
  buildMutation,
} from 'json-mutate';

// Cursor type
export enum CursorType {
  WriteCursor = 0,
  ReadForwardCursor,
  ReadBidirectionalCursor,
};

// A cursor at a given point in time.
export class Cursor {
  // Has Init() been called?
  private inited: boolean = false;
  // Do we have a state calculated at the desired timestamp?
  private ready: boolean = false;
  // Is there a reason why we aren't ready?
  private notReadyError: any;
  // Current desired timestamp
  private timestamp: Date;
  // Last snapshot before current time
  private lastSnapshot: StreamEntry;
  // Optimization: remember if we have a next snapshot.
  private nextSnapshot: StreamEntry;
  // Computed state at current timestamp
  private computedState: StateData;
  // Timestamp we computed the state at
  private _computedTimestamp: Date;
  // If we're a rewindable cursor (ReadBidirectional)
  private lastMutations: StreamEntry[];
  // If we're a write cursor
  private lastMutation: StreamEntry;
  // Last state before lastMutation
  private lastState: StateData;
  // Possibly known rate config
  private rateConfig: IRateConfig;
  // For a feed-forward cursor, subscribe to a stream of entries when fast-forwarding
  private _streamEntries: Subject<StreamEntry>;

  // Create a new cursor
  constructor(private storage: IStorageBackend,
              private cursorType: CursorType) {
    this._streamEntries = new Subject<StreamEntry>();
    this.ready = false;
    this.lastMutations = [];
  }

  get streamEntries(): Subject<StreamEntry> {
    return this._streamEntries;
  }

  // Initializes a cursor at a timestamp.
  // Timestamp should not be set for a write cursor.
  public async init(timestamp: Date = null) {
    if (this.inited) {
      throw new Error('Do not call init twice.');
    }
    this.inited = true;
    if (this.cursorType === CursorType.WriteCursor) {
      if (timestamp) {
        throw new Error('Timestamp should not be given for write cursors.');
      }
      this.ready = false;
      this.timestamp = new Date();
    } else {
      this.setTimestamp(timestamp);
    }
    await this.computeState();
  }

  public async initWithSnapshot(snap: StreamEntry) {
    if (this.cursorType === CursorType.WriteCursor) {
      throw new Error('Cannot initialize write cursor with snapshot.');
    }
    this.ready = true;
    this.timestamp = snap.timestamp;
    this.lastSnapshot = snap;
    this.copySnapshotState();
    await this.fillNextSnapshot();
  }

  public setRateConfig(config: IRateConfig) {
    this.rateConfig = config;
  }

  // Get the computed state
  get state() {
    if (!this.ready) {
      throw new Error('Computation is not ready.');
    }
    return this.computedState;
  }

  get computedTimestamp() {
    return this._computedTimestamp;
  }

  public setTimestamp(timestamp: Date) {
    if (this.cursorType === CursorType.WriteCursor) {
      return;
    }
    if (this.ready && this.timestamp.getTime() === timestamp.getTime()) {
      return;
    }
    this.ready = false;
    this.timestamp = timestamp;
    if (this.lastSnapshot &&
        this.lastSnapshot.timestamp.getTime() > timestamp.getTime()) {
      this.lastSnapshot = null;
      this.lastMutations = [];
      this.computedState = null;
    }

    if (this.computedState &&
        this.computedTimestamp.getTime() > timestamp.getTime() &&
        this.cursorType === CursorType.ReadForwardCursor) {
      this.computedState = null;
    }
  }

  public async computeState(): Promise<void> {
    if (this.ready) {
      return;
    }

    let err: any;
    try {
      await this.doComputeState();
    } catch (e) {
      err = e;
    }

    this.ready = !err;
    if (err === NoDataError && this.cursorType === CursorType.WriteCursor) {
      this.ready = true;
      this.computedState = {};
      this._computedTimestamp = new Date(this.timestamp.getTime());
      err = null;
      this.lastState = null;
    }
    this.notReadyError = err;

    if (err) {
      throw err;
    }
  }

  // Force a re-computation.
  public invalidate() {
    this.ready = false;
  }

  // Handle a state entry on a writer (keeps it up to date)
  public async handleEntry(entry: StreamEntry): Promise<void> {
    // Assert we can handle a new entry
    this.canHandleNewEntry(entry.timestamp);

    if (entry.type === StreamEntryType.StreamEntrySnapshot) {
      this.lastSnapshot = entry;
      this.copySnapshotState();
    } else {
      this.applyMutation(entry);
    }

    this._streamEntries.next(entry);
  }

  public async writeEntry(entry: StreamEntry, config: IRateConfig) {
    if (entry.type === StreamEntryType.StreamEntrySnapshot) {
      await this.writeState(entry.timestamp, entry.data, config);
      return;
    }

    let newStateData = applyMutationObject(Clone(this.state), entry.data);
    await this.writeState(entry.timestamp, newStateData, config);
  }

  public async writeState(timestamp: Date, state: StateData, config: IRateConfig): Promise<void> {
    let savedEntry: StreamEntry;
    let err: any;

    try {
      await (async () => {
        if (this.lastState && DeepEqual(this.lastState, state)) {
          return;
        }

        // Assert we can handle a new entry.
        this.canHandleNewEntry(timestamp);

        let inputState: StateData = Clone(state);
        let lastChange: Date;
        if (!this.lastMutation) {
          if (this.lastSnapshot) {
            lastChange = this.lastSnapshot.timestamp;
          }
        } else {
          lastChange = this.lastMutation.timestamp;
        }

        if (this.lastState &&
            lastChange &&
            timestamp.getTime() < lastChange.getTime()) {
          throw new Error('Cannot write entry before last change.');
        }

        if (this.lastMutation &&
            (timestamp.getTime() - this.lastMutation.timestamp.getTime()) <
            config.change_frequency) {
          let amendedMutation: StreamEntry = {
            type: StreamEntryType.StreamEntryMutation,
            timestamp: this.lastMutation.timestamp,
            data: null,
          };

          if (this.lastState && this._streamEntries.observers.length) {
            savedEntry = {
              type: StreamEntryType.StreamEntryMutation,
              timestamp: timestamp,
              data: buildMutation(Clone(this.lastState), inputState),
            };
          }

          // Calculate the new mutation
          amendedMutation.data = buildMutation(Clone(this.lastState), inputState);
          await this.storage.amendEntry(amendedMutation, this.lastMutation.timestamp);

          this.computedState = inputState;
          this._computedTimestamp = timestamp;
          this.lastMutation = amendedMutation;
          return;
        }

        // Check if we should make a new snapshot
        if (!this.lastState ||
           (timestamp.getTime() - this.lastSnapshot.timestamp.getTime()) >=
           config.keyframe_frequency) {
          let snapshot: StreamEntry = {
            type: StreamEntryType.StreamEntrySnapshot,
            data: inputState,
            timestamp: timestamp,
          };
          savedEntry = snapshot;

          await this.storage.saveEntry(snapshot);
          this.lastSnapshot = snapshot;
          this.lastMutation = null;
          try {
            this.copySnapshotState();
          } catch (e) {
            this.ready = false;
            this.computedState = null;
            throw e;
          }
          return;
        }

        // Make a new mutation
        let oldState: StateData = Clone(this.computedState);
        let newMutation: StreamEntry = {
          type: StreamEntryType.StreamEntryMutation,
          timestamp: timestamp,
          data: buildMutation(this.lastState, inputState),
        };

        savedEntry = newMutation;
        this.storage.saveEntry(newMutation);

        this.lastMutation = newMutation;
        this.lastState = oldState;
        this.computedState = inputState;
        this._computedTimestamp = timestamp;
      })();
    } catch (e) {
      err = e;
    }

    if (!err && savedEntry) {
      this._computedTimestamp = savedEntry.timestamp;
      this._streamEntries.next(savedEntry);
    }

    if (err) {
      throw err;
    }
  }

  get error() {
    return this.notReadyError;
  }

  get isReady() {
    return this.ready;
  }

  private applyMutation(mutation: StreamEntry) {
    let beforeObj: StateData;
    if (this.cursorType === CursorType.ReadBidirectionalCursor ||
        this.cursorType === CursorType.WriteCursor) {
      beforeObj = Clone(this.computedState);
    }

    let stateAfter = applyMutationObject(this.computedState, mutation.data);

    if (this.cursorType === CursorType.ReadBidirectionalCursor) {
      let beforeData = beforeObj;
      this.lastMutations.push({
        type: StreamEntryType.StreamEntryMutation,
        data: buildMutation(stateAfter, beforeData),
        timestamp: mutation.timestamp,
      });
    } else if (this.cursorType === CursorType.WriteCursor) {
      this.lastMutation = mutation;
      this.lastState = beforeObj;
    }

    this.computedState = stateAfter;
    this._computedTimestamp = mutation.timestamp;
  }

  private async doComputeState() {
    if (!this.lastSnapshot) {
      await this.fillLastSnapshot();
      await this.fillNextSnapshot();
    }

    if (this.timestamp.getTime() === this.lastSnapshot.timestamp.getTime()) {
      this.copySnapshotState();
      this.nextSnapshot = null;
    } else if (this.computedState) {
      if (this._computedTimestamp.getTime() > this.timestamp.getTime()) {
        this.rewindState();
      } else {
        if (this._streamEntries.observers.length &&
            this.nextSnapshot &&
            this.nextSnapshot.timestamp.getTime() < this.timestamp.getTime()) {
          this.lastSnapshot = this.nextSnapshot;
          this.nextSnapshot = null;
          await this.fillNextSnapshot();
          if (!this.nextSnapshot || this.nextSnapshot.timestamp.getTime() < this.timestamp.getTime()) {
            this.lastSnapshot = null;
            this.nextSnapshot = null;
            await this.fillLastSnapshot();
            await this.fillNextSnapshot();
          }
        }
        await this.fastForwardState();
      }
    } else {
      this._computedTimestamp = new Date(this.lastSnapshot.timestamp.getTime());
      this.copySnapshotState();
      await this.fastForwardState();
    }
  }

  // Throws if we cannot handle a new entry.
  private canHandleNewEntry(timestamp: Date) {
    if (this.cursorType !== CursorType.WriteCursor || !this.ready) {
      throw new Error('Cursor is not ready, or is not a write cursor.');
    }
    if (this.lastState && timestamp.getTime() < this.computedTimestamp.getTime()) {
      throw new Error('Entry is before the latest entry, we can\'t handle this.');
    }
  }

  private async fastForwardState() {
    try {
      while (this.computedTimestamp.getTime() < this.timestamp.getTime()) {
        let entry = await this.storage.getEntryAfter(this.computedTimestamp, StreamEntryType.StreamEntryAny);
        if (!entry) {
          break;
        }
        if (entry.timestamp.getTime() < this.computedTimestamp.getTime()) {
          throw new Error('Storage backend returned an entry before requested time.');
        }
        if (entry.timestamp.getTime() > this.timestamp.getTime()) {
          if (entry.type === StreamEntryType.StreamEntrySnapshot) {
            this.nextSnapshot = entry;
          }
          break;
        }
        this._streamEntries.next(entry);
        if (entry.type === StreamEntryType.StreamEntryMutation) {
          this.applyMutation(entry);
        } else if (entry.type === StreamEntryType.StreamEntrySnapshot) {
          this.lastSnapshot = entry;
          this.nextSnapshot = null;
          this.copySnapshotState();
          await this.fillNextSnapshot();
        }
      }
    } catch (e) {
      this.computedState = null;
      this.ready = false;
      throw e;
    }
  }

  private copySnapshotState() {
    this.computedState = Clone(this.lastSnapshot.data);
    this._computedTimestamp = new Date(this.lastSnapshot.timestamp.getTime());
    this.lastMutation = null;
    this.lastState = this.computedState;
    this.lastMutations = [];
  }

  private async fillLastSnapshot() {
    let data = await this.storage.getSnapshotBefore(this.timestamp);
    if (!data) {
      throw NoDataError;
    }
    if (data.type !== StreamEntryType.StreamEntrySnapshot) {
      throw new Error('Storage backend didn\'t return a snapshot for getSnapshotBefore()');
    }
    if (data.timestamp.getTime() > this.timestamp.getTime()) {
      throw new Error('Storage backend returned a snapshot after the requested timestamp.');
    }
    this.lastSnapshot = data;
    if (this.lastSnapshot && data.timestamp.getTime() === this.lastSnapshot.timestamp.getTime()) {
      // We're using the last one again
      return;
    }
    this.lastMutations = [];
    this.computedState = null;
  }

  private async fillNextSnapshot() {
    if (!this.lastSnapshot) {
      return;
    }

    if (this.rateConfig) {
      let expectedNext = new Date(this.lastSnapshot.timestamp.getTime());
      expectedNext.setMilliseconds(expectedNext.getMilliseconds() + this.rateConfig.keyframe_frequency);
      if (expectedNext.getTime() > (new Date()).getTime()) {
        this.nextSnapshot = null;
        return;
      }
    }

    let snap = await this.storage.getEntryAfter(this.lastSnapshot.timestamp, StreamEntryType.StreamEntrySnapshot);
    if (snap && snap.type !== StreamEntryType.StreamEntrySnapshot) {
      throw new Error('Storage backend returned the wrong entry type.');
    }
    this.nextSnapshot = snap;
  }

  // Rewinds the state
  private rewindState() {
    let idx = this.lastMutations.length - 1;
    let err: any;
    try {
      // TODO: trim lastMutations to idx max
      if (idx < 0 || this.lastMutations[0].timestamp.getTime() > this.timestamp.getTime()) {
        this.copySnapshotState();
        if (idx !== 0) {
          idx = -1;
        }
        return;
      }

      if (this.lastMutations[idx].timestamp.getTime() < this.timestamp.getTime()) {
        return;
      }

      // Apply mutations backwards until next mutation is before target.
      while (!(this.lastMutations[idx].timestamp.getTime() < this.timestamp.getTime())) {
        let mutation = this.lastMutations[idx];
        let newObj = applyMutationObject(this.computedState, mutation.data);
        this.computedState = newObj;
        idx--;
      }
    } catch (e) {
      err = e;
    }

    if (err) {
      this.computedState = null;
    }

    if (this.lastMutations.length) {
      if (idx < 0) {
        this.lastMutations = [];
      } else {
        this.lastMutations = this.lastMutations.slice(0, idx + 1);
      }
    }

    if (err) {
      throw err;
    }
  }
}
