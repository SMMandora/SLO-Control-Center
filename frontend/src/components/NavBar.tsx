import { NavLink } from "react-router-dom";

const TABS = [
  { to: "/", label: "Overview", end: true },
  { to: "/services", label: "Services" },
  { to: "/incidents", label: "Incidents" },
  { to: "/alerts", label: "Alerts" },
  { to: "/traces", label: "Traces" },
  { to: "/logs", label: "Logs" },
  { to: "/capacity", label: "Capacity" },
  { to: "/runbooks", label: "Runbooks" },
];

export function NavBar() {
  return (
    <nav className="flex gap-1 px-4">
      {TABS.map((t) => (
        <NavLink
          key={t.to}
          to={t.to}
          end={t.end}
          className={({ isActive }) =>
            `px-3 py-2 text-sm rounded-t-md ${
              isActive
                ? "text-white border-b-2 border-violet-400"
                : "text-slate-400 hover:text-slate-200"
            }`
          }
        >
          {t.label}
        </NavLink>
      ))}
    </nav>
  );
}
