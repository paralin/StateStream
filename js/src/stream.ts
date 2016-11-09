import { IConfig, DefaultStreamConfig } from './config';
import { IStorageBackend } from './backend';
import { Cursor, CursorType } from './cursor';
import { StreamEntry, StateData } from './entry';

export class Stream {
  private _writeCursor: Cursor;

  constructor(private _storage: IStorageBackend,
              private _config: IConfig = null) {
    if (!this._config) {
      this._config = DefaultStreamConfig();
    }
  }

  public get config() {
    return this._config;
  }

  public get storage() {
    return this._storage;
  }

  public resetWriter() {
    this._writeCursor = null;
  }

  public disableAmends() {
    this.config.record_rate.change_frequency = 0;
    if (this._writeCursor) {
      this._writeCursor.setRateConfig(this.config.record_rate);
    }
  }

  public async initWriter() {
    if (this._writeCursor) {
      return;
    }

    let cursor = this.buildCursor(CursorType.WriteCursor);
    await cursor.init();
    this._writeCursor = cursor;
  }

  public get writeCursor(): Promise<Cursor> {
    return (async () => {
      if (!this._writeCursor) {
        await this.initWriter();
      }
      return this._writeCursor;
    })();
  }

  public async writeState(timestamp: Date, state: StateData) {
    let cursor = await this.writeCursor;
    await cursor.writeState(timestamp, state, this.config.record_rate);
  }

  public async writeEntry(entry: StreamEntry) {
    let cursor = await this.writeCursor;
    await cursor.writeEntry(entry, this.config.record_rate);
  }

  public buildCursor(cursorType: CursorType) {
    return new Cursor(this._storage, cursorType);
  }
}
