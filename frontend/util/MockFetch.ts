export type FetchFn = (
  input: RequestInfo,
  init?: RequestInit | undefined
) => Promise<Response>;

export interface FetchParams {
  input: RequestInfo;
  init?: RequestInit | undefined;
}

export default class MockFetch {
  callStack: FetchParams[];
  resultStack: Response[];
  globalFetch: FetchFn;
  headers: Headers;

  constructor(headers?: HeadersInit | undefined) {
    this.callStack = [];
    this.resultStack = [];
    this.globalFetch = global.fetch;
    this.headers = new Headers(headers);
  }

  start() {
    this.globalFetch = global.fetch;
    const mockFetch = this;
    global.fetch = jest.fn(
      (
        input: RequestInfo,
        init?: RequestInit | undefined
      ): Promise<Response> => {
        return mockFetch.call(input, init);
      }
    );
  }

  finish() {
    global.fetch = this.globalFetch;
    if (this.callStack.length > 0) {
      throw new Error(`expected ${this.callStack.length} more calls`);
    }
    this.callStack = [];
    this.resultStack = [];
  }

  expect(input: RequestInfo, init?: RequestInit | undefined) {
    this.callStack.push({ input, init });
    return this;
  }

  returns(resp: Response) {
    this.resultStack.push(resp);
    return this;
  }

  call(input: RequestInfo, init?: RequestInit | undefined): Promise<Response> {
    let result = this.resultStack.pop();
    let expected = this.callStack.pop();
    expect(result).not.toEqual(undefined);
    expect(expected).not.toEqual(undefined);
    expect(input).toEqual(expected?.input);
    expect(init).toEqual(expected?.init);
    return Promise.resolve(result || ({} as Response));
  }
}
