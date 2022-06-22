import "~/frontend/pages/admin/styles.css";
import * as api from "@hrry.me/api";
import { logs, RequestLog } from "@hrry.me/api/logs";
import LoginManager from "~/frontend/util/LoginManager";
import {
  TOKEN_KEY,
  Role,
  storeToken,
  deleteToken,
  setCookie,
} from "~/frontend/api/auth";
import { clearCookie } from "~/frontend/util/cookies";
import { SECOND } from "~/frontend/constants";
import { millisecondsToStr } from "~/frontend/util/time";

const main = () => {
  let loginManager = new LoginManager({
    interval: 5 * 60 * SECOND,
    clearToken: () => {
      deleteToken();
      clearCookie(TOKEN_KEY);
    },
  });

  let inviteTable = new Table(
    "invite-list",
    (_index: number, _offset: number): Promise<any[][]> => {
      return api.invites().then((res: api.InviteList) => {
        let rows = [];
        if (res.invites == null) return [];
        res.invites.sort((a: api.InviteURL, b: api.InviteURL): number => {
          return (
            new Date(a.expires_at).getTime() - new Date(b.expires_at).getTime()
          );
        });
        for (let inv of res.invites) {
          rows.push(inviteRow(inv));
        }
        return rows;
      });
    }
  );
  inviteTable.header(["url", "email", "name", "expires at", "roles"]);
  let inviteForm = document.getElementById("invite-source") as HTMLFormElement;
  if (loginManager.isLoggedIn()) {
    inviteForm.style.visibility = "visible";
  } else {
    // redirect to home if not logged in
    window.location.pathname = "/";
    return;
  }
  handleInvitationCreation(inviteTable, inviteForm);
  inviteTable.render();

  document.addEventListener("tokenChange", (ev: TokenChangeEvent) => {
    const e = ev.detail;
    if (e.action == "login") {
      storeToken(e.token);
      setCookie(e.token);
    } else {
      if (!loginManager.isLoggedIn()) {
        return;
      }
    }
  });

  let infoContainer = document.getElementById("server-info");
  api.runtimeInfo().then((info: api.RuntimeInfo) => {
    if (infoContainer == null) {
      return;
    }
    let elements = [
      ...keyPairElem("name", info.name),
      ...keyPairElem("age", info.age),
      ...keyPairElem("uptime", info.uptime),
      ...keyPairElem("birthday", info.birthday),
      ...keyPairElem("debug", info.debug),
      ...keyPairElem("GOOS", info.build.GOOS),
      ...keyPairElem("GOARCH", info.build.GOARCH),
      ...keyPairElem("vcs", info.build["vcs"]),
      ...keyPairElem("commit", info.build["vcs.revision"]),
      ...keyPairElem("commit.date", new Date(info.build["vcs.time"])),
    ];
    let dl: HTMLDListElement = document.createElement("dl");
    for (let elem of elements) {
      dl.appendChild(elem);
    }
    infoContainer.appendChild(dl);
  });
  /*
  let table = new Table("logs", logsPaginator);
  table.header([
    "id", "method", "status", "ip",
    "uri", "referer", "user agent", "latency",
    "requested at",
  ]);
  table.render();
  */
};

const createInviteRequest = (data: FormData): api.InviteRequest => {
  let expires = (data.get("expires") as string) || undefined;
  let email = (data.get("email") as string) || undefined;
  let receiverName = (data.get("recipient-name") as string) || undefined;
  let ttl = (data.get("ttl") as string) || undefined;
  let roles = (data.get("roles") as string) || undefined;

  let request: api.InviteRequest = {};
  if (expires) {
    let ex = new Date(expires);
    // timeout in nanoseconds
    request.timeout = (ex.getTime() - Date.now()) * 1e6;
  }
  if (ttl) request.ttl = parseInt(ttl);
  if (email) request.email = email;
  if (receiverName) request.receiver_name = receiverName;
  if (roles) request.roles = roles.split(",");
  return request;
};

const inviteRow = (inv: api.InviteURL): string[] => {
  let url = location.origin;
  if (inv.path[0] != "/") url += "/";
  if (inv.roles == null) inv.roles = [];
  let d = new Date(inv.expires_at);
  let diff = d.getTime() - Date.now();
  let roles: Role[] =
    inv.roles == null || inv.roles.length == 0 ? [Role.Default] : inv.roles;
  return [
    `${url}${inv.path}`,
    inv.email || "",
    inv.receiver_name,
    `${millisecondsToStr(diff)}`,
    roles
      .map((v: Role) => {
        return Role[v].toLowerCase();
      })
      .join(", "),
  ];
};

const handleInvitationCreation = (table: Table, form: HTMLFormElement) => {
  form.addEventListener("submit", (ev: SubmitEvent) => {
    ev.preventDefault();
    let target = ev.target as HTMLFormElement;
    let data = new FormData(target);
    let request = createInviteRequest(data);
    api.invite(request).then((invite: api.InviteURL) => {
      table.pushRow(inviteRow(invite));
    });
  });
};

const logsPaginator = async (
  limit: number,
  offset: number
): Promise<any[][]> => {
  return logs({ limit: limit, offset: offset, reverse: true }).then(
    (logs: RequestLog[]) =>
      logs.map((log: RequestLog, index: number) => [
        log.id,
        log.method,
        log.status,
        log.ip,
        log.uri,
        log.referer || "-",
        log.user_agent,
        `${log.latency / 1e6} ms`,
        log.requested_at,
      ])
  );
};

const keyPairElem = (name: string, value: any): [HTMLElement, HTMLElement] => {
  let k = document.createElement("dt");
  k.innerText = name;
  let v = document.createElement("dd");
  v.innerText = `${value}`;
  return [k, v];
};

const keyElement = (name: string): HTMLElement => {
  let el = document.createElement("dt");
  el.innerText = name;
  return el;
};

const valueElement = (value: any): HTMLElement => {
  let el = document.createElement("dd");
  el.innerText = `${value}`;
  return el;
};

type Paginator = (index: number, offset: number) => Promise<any[][]>;

class Table {
  container: HTMLElement;
  table: HTMLTableElement;
  paginator: Paginator;

  private thead: HTMLTableSectionElement;
  private tbody: HTMLTableSectionElement;
  private headerNames: string[];

  constructor(containerID: string, paginator: Paginator) {
    let container = document.getElementById(containerID);
    if (container == null) {
      throw new Error(`container id "${containerID}" not found`);
    }
    this.paginator = paginator;
    this.container = container;
    this.headerNames = [];

    this.table = document.createElement("table");
    this.thead = document.createElement("thead");
    this.tbody = document.createElement("tbody");
    this.table.setAttribute("cellspacing", "0");
    this.table.setAttribute("cellpadding", "0");
    this.table.appendChild(this.thead);
    this.table.appendChild(this.tbody);
  }

  clear() {
    this.table.removeChild(this.thead);
    this.table.removeChild(this.tbody);
    this.thead = document.createElement("thead");
    this.tbody = document.createElement("tbody");
    this.table.appendChild(this.thead);
    this.table.appendChild(this.tbody);
  }

  render() {
    this.paginator(200, 0).then((rows: any[][]) => {
      for (let r of rows) {
        this.addRow(r);
      }
      this.container.appendChild(this.table);
    });
  }

  header(header: string[]) {
    let h = document.createElement("tr");
    this.headerNames = header;
    for (let i = 0; i < header.length; i++) {
      let el = document.createElement("th");
      el.setAttribute("scope", "col");
      el.innerText = header[i];

      if (i < header.length - 1) {
        let handle = document.createElement("div");
        handle.className = "table-resize-handle";
        handle.addEventListener("mousedown", (ev: MouseEvent) => {});
        el.appendChild(handle);
        addColResizeHandlers(handle);
      }

      h.appendChild(el);
    }
    this.thead.appendChild(h);
  }

  private createRow(row: any[]): HTMLTableRowElement {
    let tr = document.createElement("tr");
    let i = 0;
    for (let val of row) {
      let el = document.createElement("td");
      el.setAttribute("data-label", this.headerNames[i]);
      el.innerText = `${val}`;
      tr.appendChild(el);
      i++;
    }
    return tr;
  }

  addRow(row: any[]) {
    this.tbody.appendChild(this.createRow(row));
  }

  pushRow(row: any[]) {
    this.tbody.insertBefore(this.createRow(row), this.tbody.firstChild);
  }

  body(rows: any[][]) {
    for (let row of rows) {
      this.addRow(row);
    }
  }
}

// TODO this is horrible, plz fix
// Taken from https://www.brainbell.com/javascript/making-resizable-table-js.html
const addColResizeHandlers = (el: HTMLElement) => {
  let pageX: number;
  let curCol: HTMLElement | null;
  let nextCol: HTMLElement | null;
  let curColWidth: number;
  let nextColWidth: number;
  let handlerSet = false;

  const mouseMove = (ev: MouseEvent) => {
    if (curCol) {
      let diffX = ev.pageX - pageX;
      if (nextCol) {
        nextCol.style.width = `${nextColWidth - diffX}px`;
      }
      curCol.style.width = `${curColWidth + diffX}px`;
    }
  };

  el.addEventListener("mousedown", (ev: MouseEvent) => {
    let target = ev.target as HTMLElement;
    if (target.parentElement == null) {
      return;
    }
    curCol = target.parentElement;
    nextCol = curCol.nextElementSibling as HTMLElement | null;
    pageX = ev.pageX;
    curColWidth = curCol.offsetWidth;
    if (nextCol) {
      nextColWidth = nextCol.offsetWidth;
    }
    console.log("adding mouseMove event handler");
    document.addEventListener("mousemove", mouseMove);
    handlerSet = true;
  });

  document.addEventListener("mouseup", (ev: MouseEvent) => {
    curCol = null;
    nextCol = null;
    curColWidth = 0;
    nextColWidth = 0;

    if (handlerSet) {
      console.log("removing mouseMove event handler");
      document.removeEventListener("mousemove", mouseMove);
      handlerSet = false;
    }
  });
};

main();
