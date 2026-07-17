// Hybrid Logical Clock, identisches Format wie im Go-Server:
//   "<ms:15>:<counter:5>:<nodeId>"  -> String-Sort == Zeit-Sort.
// nodeId is a stable, per-device randomly generated ID.

const KEY_NODE = 'obh.nodeId';

function nodeId(): string {
  let id = localStorage.getItem(KEY_NODE);
  if (!id) {
    id = crypto.randomUUID().slice(0, 8);
    localStorage.setItem(KEY_NODE, id);
  }
  return id;
}

const pad = (n: number, w: number) => String(n).padStart(w, '0');

export class HLC {
  private lastMs = 0;
  private counter = 0;
  readonly node = nodeId();

  now(): string {
    const pt = Date.now();
    if (pt > this.lastMs) {
      this.lastMs = pt;
      this.counter = 0;
    } else {
      this.counter++;
    }
    return this.fmt(this.lastMs, this.counter);
  }

  recv(remote: string) {
    const [rms, rc] = parse(remote);
    const pt = Date.now();
    if (pt > this.lastMs && pt > rms) {
      this.lastMs = pt;
      this.counter = 0;
    } else if (rms > this.lastMs) {
      this.lastMs = rms;
      this.counter = rc + 1;
    } else if (this.lastMs > rms) {
      this.counter++;
    } else {
      this.counter = Math.max(this.counter, rc) + 1;
    }
  }

  private fmt(ms: number, c: number) {
    return `${pad(ms, 15)}:${pad(c, 5)}:${this.node}`;
  }
}

function parse(s: string): [number, number] {
  const p = s.split(':');
  return [Number(p[0] ?? 0), Number(p[1] ?? 0)];
}

export const compare = (a: string, b: string) => (a < b ? -1 : a > b ? 1 : 0);
export const hlc = new HLC();
