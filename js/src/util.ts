export function toPromise<T>(val: T | Promise<T>): Promise<T> {
  if (!val || typeof val !== 'object' || val.constructor !== Promise) {
    return Promise.resolve<T>(val);
  } else {
    return <Promise<T>>val;
  }
}
