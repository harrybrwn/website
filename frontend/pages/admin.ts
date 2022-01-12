import "../styles/font.css";
import "../styles/admin.css";
import * as api from "../api";

const main = () => {
  let table = new Table("logs", logsPaginator);
  table.header([
    "id",
    "method",
    "status",
    "ip",
    "uri",
    "referer",
    "user agent",
    "latency",
    "requested at",
  ]);
  table.render();
};

const logsPaginator = async (
  limit: number,
  offset: number
): Promise<any[][]> => {
  return api
    .logs({ limit: limit, offset: offset, reverse: true })
    .then((logs: api.RequestLog[]) =>
      logs.map((log: api.RequestLog, index: number) => [
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
    this.table.setAttribute("cellspacing", "0");
    this.table.setAttribute("cellpadding", "0");
    this.thead = document.createElement("thead");
    this.tbody = document.createElement("tbody");
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

  addRow(row: any[]) {
    let tr = document.createElement("tr");
    let i = 0;
    for (let val of row) {
      let el = document.createElement("td");
      el.setAttribute("data-label", this.headerNames[i]);
      el.innerText = `${val}`;
      tr.appendChild(el);
      i++;
    }
    this.tbody.appendChild(tr);
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
      console.log("removeing mouseMove event handler");
      document.removeEventListener("mousemove", mouseMove);
      handlerSet = false;
    }
  });
};

main();
