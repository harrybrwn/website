import { RequestLog, logs } from "../api";

const main = () => {
  let table = new Table("logs");
  logs({ limit: 200, reverse: true }).then((logs) => {
    displayLogs(table, logs);
  });
};

const displayLogs = (table: Table, logs: RequestLog[]) => {
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
  for (let l of logs) {
    table.addRow([
      l.id,
      l.method,
      l.status,
      l.ip,
      l.uri,
      l.referer,
      l.user_agent,
      `${l.latency / 1e6} ms`,
      l.requested_at,
    ]);
  }
  table.render();
};

class Table {
  container: HTMLElement;
  table: HTMLTableElement;

  constructor(containerID: string) {
    this.container =
      document.getElementById(containerID) || document.createElement("section");
    this.table = document.createElement("table");
  }

  render() {
    this.container.appendChild(this.table);
  }

  header(header: string[]) {
    let h = document.createElement("tr");
    for (let name of header) {
      let el = document.createElement("th");
      el.innerText = name;
      h.appendChild(el);
    }
    this.table.appendChild(h);
  }

  addRow(row: any[]) {
    let tr = document.createElement("tr");
    for (let val of row) {
      let el = document.createElement("td");
      el.innerText = `${val}`;
      tr.appendChild(el);
    }
    this.table.appendChild(tr);
  }

  body(rows: any[][]) {
    for (let row of rows) {
      this.addRow(row);
    }
  }
}

main();
