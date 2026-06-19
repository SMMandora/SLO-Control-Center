import { Outlet } from "react-router-dom";
import { NavBar } from "./NavBar";

export function Layout() {
  return (
    <div className="min-h-screen bg-slate-900 text-slate-200">
      <header className="flex items-center justify-between border-b border-slate-800 px-4 pt-3">
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2 pb-3">
            <span className="text-violet-400 text-lg">◎</span>
            <span className="font-semibold text-white">SLO Control Center</span>
          </div>
          <NavBar />
        </div>
        <div className="flex items-center gap-2 pb-3 text-xs text-slate-400">
          <span className="rounded bg-slate-800 px-2 py-1">Production</span>
          <span className="rounded bg-slate-800 px-2 py-1">Last 28 Days</span>
        </div>
      </header>
      <main>
        <Outlet />
      </main>
    </div>
  );
}
