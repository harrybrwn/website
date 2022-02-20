import MockFetch from "~/frontend/util/MockFetch";
import { mockToken } from "~/frontend/util/mocks";
import { Token } from "./auth";
import { invite, InviteRequest } from "./index";

describe("invitation api", () => {
  let mockFetch: MockFetch;
  let token: Token;

  beforeEach(() => {
    mockFetch = new MockFetch();
    mockFetch.start();
    token = mockToken();
  });

  afterEach(() => {
    mockFetch.finish();
    localStorage.clear();
  });

  describe("create", () => {
    let req: InviteRequest = { ttl: 10 };
    let path: string;
    beforeEach(() => {
      path = "/invite/1234";
      mockFetch
        .expect("/api/invite/create", {
          method: "POST",
          headers: {
            Authorization: `${token.type} ${token.token}`,
            "Content-Type": "application/json",
            Accept: "application/json",
          },
          body: JSON.stringify(req),
        })
        .returns({
          ok: true,
          json() {
            return Promise.resolve({ path: path });
          },
        } as Response);
    });
    test("basic call", async () => {
      let inv = await invite(req);
      expect(inv).toBeTruthy();
      expect(inv.path).toBe(path);
    });
  });
});
