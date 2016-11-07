export enum StreamEntryType {
  StreamEntrySnapshot = 0,
  StreamEntryMutation,
  StreamEntryAny,
}

export type StateData = { [key: string]: any };

export type StreamEntry = {
  timestamp: Date;
  type: StreamEntryType;
  data: StateData;
}

// Come up with a better way to do this.
export function Clone(inp: any): any {
  return JSON.parse(JSON.stringify(inp));
}

export function DeepEqual(inp: any, inpb: any): boolean {
  return JSON.stringify(inp) === JSON.stringify(inpb);
}
