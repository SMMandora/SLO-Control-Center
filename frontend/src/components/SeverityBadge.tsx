const COLORS: Record<string, string> = {
  page: "bg-rose-700",
  ticket: "bg-amber-700",
  warn: "bg-yellow-700",
  error: "bg-rose-700",
};

export function SeverityBadge({ severity }: { severity: string }) {
  return (
    <span className={`text-xs px-2 py-0.5 rounded text-white ${COLORS[severity] ?? "bg-slate-600"}`}>
      {severity || "—"}
    </span>
  );
}
