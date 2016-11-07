import { StreamEntry, StreamEntryType, Clone } from './entry';

export function mockTime(offset: number) {
  let base = new Date(1478492726987);
  base.setSeconds(base.getSeconds() + offset);
  return base;
}

export function cloneArr(ents: StreamEntry[]) {
  let res: StreamEntry[] = [];
  for (let element of ents) {
    res.push({
      timestamp: element.timestamp,
      type: element.type,
      data: Clone(element.data),
    });
  }
  return res;
}

export const sampleData: StreamEntry[] = [
  {
    type: StreamEntryType.StreamEntrySnapshot,
    timestamp: mockTime(-10),
    data: {hello: 'world'},
  },
  {
    type: StreamEntryType.StreamEntryMutation,
    timestamp: mockTime(-9),
    data: {hello: {$set: {there: 1}}},
  },
  {
    type: StreamEntryType.StreamEntryMutation,
    timestamp: mockTime(-8),
    data: {hello: {there: 2}},
  },
  {
    type: StreamEntryType.StreamEntryMutation,
    timestamp: mockTime(-7),
    data: {hello: null},
  },
  {
    type: StreamEntryType.StreamEntrySnapshot,
    timestamp: mockTime(-6),
    data: {hello: true},
  },
  {
    type: StreamEntryType.StreamEntryMutation,
    timestamp: mockTime(-5),
    data: {goodbye: 4},
  },
];
