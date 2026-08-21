import { useCallback, useEffect, useMemo, useState } from "react";

type Identity = {
  node_id: string;
  hostname: string;
  hits: number;
  remote_addr: string;
  time: string;
};

type Probe = Identity & { latency_ms: number; ok: boolean };

type Toast = { id: number; text: string };

function formatClock(d: Date): string {
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
}

export default function App() {
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [probes, setProbes] = useState<Probe[]>([]);
  const [loading, setLoading] = useState(false);
  const [online, setOnline] = useState(false);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [clock, setClock] = useState(() => formatClock(new Date()));

  const pushToast = useCallback((text: string) => {
    const id = Date.now();
    setToasts((prev) => [...prev, { id, text }]);
    window.setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 5000);
  }, []);

  const probe = useCallback(
    async (silent = false) => {
      if (!silent) setLoading(true);
      const started = performance.now();
      try {
        const res = await fetch("/api/v1/identity");
        const body = await res.json();
        if (!res.ok) {
          throw new Error(body?.error?.message || `HTTP ${res.status}`);
        }
        const data = body.data as Identity;
        const latency = Math.round(performance.now() - started);
        setIdentity(data);
        setOnline(true);
        setProbes((prev) => [{ ...data, latency_ms: latency, ok: true }, ...prev].slice(0, 8));
      } catch (err) {
        setOnline(false);
        const text = err instanceof Error ? err.message : "探测失败";
        setProbes((prev) =>
          [
            {
              node_id: "—",
              hostname: "—",
              hits: 0,
              remote_addr: "—",
              time: formatClock(new Date()),
              latency_ms: Math.round(performance.now() - started),
              ok: false,
            },
            ...prev,
          ].slice(0, 8),
        );
        if (!silent) pushToast(text);
      } finally {
        if (!silent) setLoading(false);
      }
    },
    [pushToast],
  );

  useEffect(() => {
    void probe(true);
    const poll = window.setInterval(() => void probe(true), 3000);
    const tick = window.setInterval(() => setClock(formatClock(new Date())), 1000);
    return () => {
      window.clearInterval(poll);
      window.clearInterval(tick);
    };
  }, [probe]);

  const statusLabel = useMemo(() => {
    if (online && identity) return "ONLINE";
    if (loading) return "PROBING";
    return "OFFLINE";
  }, [online, identity, loading]);

  return (
    <div className="relative min-h-screen w-full overflow-x-hidden bg-void text-fog">
      <div className="noise absolute inset-0" />
      <div className="pointer-events-none absolute -left-24 top-[-8rem] h-[28rem] w-[28rem] rounded-full bg-phosphor/10 blur-3xl" />
      <div className="pointer-events-none absolute right-[-6rem] bottom-[-8rem] h-[24rem] w-[24rem] rounded-full bg-amber/10 blur-3xl" />

      <div className="relative w-full px-4 py-5 xs:px-6 md:px-10 md:py-8">
        <header className="rise flex w-full flex-col gap-4 border-b border-line pb-5 md:flex-row md:items-end md:justify-between">
          <div>
            <p className="font-mono text-xs tracking-[0.35em] text-phosphor">SIMPLEFRP · CABLE LANDING</p>
            <h1 className="mt-2 font-display text-3xl font-extrabold tracking-tight text-fog md:text-5xl">
              内网节点观察窗
            </h1>
            <p className="mt-2 max-w-none text-sm text-mist md:text-base">
              你现在看到的页面来自内网 Demo 服务。若地址栏是公网访客端口，说明 TCP 隧道、多路复用与转发均已接通。
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <div className="rounded-lg border border-line bg-steel px-4 py-2">
              <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-mist">Beijing Time</p>
              <p className="font-mono text-lg text-phosphor">{clock}</p>
            </div>
            <div
              className={`flex items-center gap-2 rounded-full border px-3 py-2 ${
                online ? "border-phosphor/40 bg-phosphor-dim" : "border-danger/40 bg-danger/10"
              }`}
            >
              <span className={`h-2.5 w-2.5 rounded-full ${online ? "live-dot bg-phosphor" : "bg-danger"}`} />
              <span className="font-mono text-xs tracking-[0.18em]">{statusLabel}</span>
            </div>
          </div>
        </header>

        <div className="mt-6 grid w-full grid-cols-1 gap-5 md:grid-cols-5">
          <section className="rise md:col-span-2 rounded-2xl border border-line bg-steel p-5 transition hover:border-phosphor/50 md:p-6" style={{ animationDelay: "80ms" }}>
            <p className="font-mono text-xs tracking-[0.28em] text-amber">IDENTITY PLATE</p>
            <h2 className="mt-3 font-display text-2xl font-bold">{identity?.node_id ?? "等待探测"}</h2>
            <dl className="mt-5 grid grid-cols-1 gap-3 xs:grid-cols-2">
              <Meta label="Hostname" value={identity?.hostname ?? "—"} />
              <Meta label="Hits" value={identity ? String(identity.hits) : "—"} />
              <Meta label="Peer" value={identity?.remote_addr ?? "—"} />
              <Meta label="Last probe" value={identity?.time ?? "—"} />
            </dl>
            <button
              type="button"
              onClick={() => void probe(false)}
              disabled={loading}
              className="mt-6 w-full rounded-xl border border-phosphor/40 bg-phosphor-dim px-4 py-3 font-display text-sm font-semibold tracking-wide text-phosphor transition hover:bg-phosphor hover:text-void disabled:cursor-wait disabled:opacity-60"
            >
              {loading ? "正在探测…" : "重新探测身份"}
            </button>
          </section>

          <section className="rise md:col-span-3 overflow-hidden rounded-2xl border border-line bg-steel p-5 md:p-6" style={{ animationDelay: "160ms" }}>
            <p className="font-mono text-xs tracking-[0.28em] text-amber">SONAR DECK</p>
            <div className="relative mt-4 h-56 w-full md:h-64">
              <div className="absolute left-1/2 top-1/2 h-40 w-40 -translate-x-1/2 -translate-y-1/2 sonar-ring" />
              <div className="absolute left-1/2 top-1/2 h-56 w-56 -translate-x-1/2 -translate-y-1/2 sonar-ring" style={{ animationDelay: "2s" }} />
              <div className="absolute left-1/2 top-1/2 h-72 w-72 -translate-x-1/2 -translate-y-1/2 sonar-ring" style={{ animationDelay: "4s" }} />
              <svg viewBox="0 0 640 200" className="absolute inset-0 h-full w-full">
                <path
                  d="M 40 70 C 160 70, 200 130, 320 130 S 480 70, 600 70"
                  fill="none"
                  stroke="#2A3A4A"
                  strokeWidth="2"
                />
                <path
                  d="M 40 70 C 160 70, 200 130, 320 130 S 480 70, 600 70"
                  fill="none"
                  stroke="#7CFFB2"
                  strokeWidth="1.5"
                  strokeDasharray="6 10"
                  opacity="0.7"
                />
                <circle className="packet" r="5" fill="#7CFFB2" />
                <circle className="packet packet-b" r="4" fill="#F5B942" />
                <circle className="packet packet-c" r="3" fill="#C5D0DA" />
                <text x="28" y="52" fill="#7A8B9A" fontSize="11" fontFamily="Fragment Mono, monospace">
                  CLIENT
                </text>
                <text x="286" y="168" fill="#7A8B9A" fontSize="11" fontFamily="Fragment Mono, monospace">
                  MUX / POOL
                </text>
                <text x="548" y="52" fill="#7A8B9A" fontSize="11" fontFamily="Fragment Mono, monospace">
                  VISITOR
                </text>
              </svg>
            </div>
            <p className="mt-2 font-mono text-xs text-mist">
              单条 TCP 长连接上的多路复用流。光点互不阻塞，对应并发访客。
            </p>
          </section>
        </div>

        <section className="rise mt-5 w-full rounded-2xl border border-line bg-steel p-4 md:p-6" style={{ animationDelay: "240ms" }}>
          <div className="mb-4 flex items-center justify-between gap-3">
            <p className="font-mono text-xs tracking-[0.28em] text-amber">PROBE LOG</p>
            <p className="text-xs text-mist">最近 8 次身份探测 · 延迟含隧道往返</p>
          </div>
          <div className="w-full overflow-x-auto">
            <table className="w-full min-w-[640px] border-collapse text-left">
              <thead>
                <tr className="border-b border-line font-mono text-[11px] uppercase tracking-[0.16em] text-mist">
                  <th className="py-2 pr-4 font-medium">Time</th>
                  <th className="py-2 pr-4 font-medium">Node</th>
                  <th className="py-2 pr-4 font-medium">Hostname</th>
                  <th className="py-2 pr-4 font-medium">Hits</th>
                  <th className="py-2 pr-4 font-medium">Latency</th>
                  <th className="py-2 font-medium">Result</th>
                </tr>
              </thead>
              <tbody>
                {probes.length === 0 ? (
                  <tr>
                    <td className="py-6 text-sm text-mist" colSpan={6}>
                      尚无探测记录，正在连接内网节点…
                    </td>
                  </tr>
                ) : (
                  probes.map((row, idx) => (
                    <tr key={`${row.time}-${idx}`} className="border-b border-line/60 font-mono text-sm">
                      <td className="py-3 pr-4">{row.time}</td>
                      <td className="py-3 pr-4 text-phosphor">{row.node_id}</td>
                      <td className="py-3 pr-4">{row.hostname}</td>
                      <td className="py-3 pr-4">{row.hits || "—"}</td>
                      <td className="py-3 pr-4">{row.latency_ms} ms</td>
                      <td className={row.ok ? "py-3 text-phosphor" : "py-3 text-danger"}>
                        {row.ok ? "THROUGH" : "FAIL"}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>
      </div>

      <div className="pointer-events-none fixed right-4 top-4 z-50 flex w-[min(100%,20rem)] flex-col gap-2 xs:right-6">
        {toasts.map((t) => (
          <div
            key={t.id}
            className="pointer-events-auto flex items-start justify-between gap-3 rounded-xl border border-danger/40 bg-steel-2 px-4 py-3 shadow-lg"
          >
            <p className="text-sm text-fog">{t.text}</p>
            <button
              type="button"
              aria-label="关闭"
              className="text-mist transition hover:text-fog"
              onClick={() => setToasts((prev) => prev.filter((x) => x.id !== t.id))}
            >
              ×
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-line bg-void/60 px-3 py-3">
      <dt className="font-mono text-[10px] uppercase tracking-[0.2em] text-mist">{label}</dt>
      <dd className="mt-1 break-all font-mono text-sm text-fog">{value}</dd>
    </div>
  );
}
