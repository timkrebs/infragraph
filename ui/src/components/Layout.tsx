import { NavLink, Outlet, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Network,
  Server,
  Zap,
  Radio,
  Settings,
  Activity,
  ChevronRight,
} from 'lucide-react';
import { useState } from 'react';

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/resources', icon: Server, label: 'Resources' },
  { to: '/graph', icon: Network, label: 'Graph' },
  { to: '/impact', icon: Zap, label: 'Impact Analysis' },
  { to: '/collectors', icon: Radio, label: 'Collectors' },
  { to: '/settings', icon: Settings, label: 'Settings' },
];

function breadcrumbsFromPath(pathname: string) {
  const segs = pathname.split('/').filter(Boolean);
  const crumbs = [{ label: 'InfraGraph', to: '/' }];
  let path = '';
  for (const seg of segs) {
    path += '/' + seg;
    crumbs.push({ label: decodeURIComponent(seg), to: path });
  }
  return crumbs;
}

export default function Layout() {
  const location = useLocation();
  const breadcrumbs = breadcrumbsFromPath(location.pathname);
  const [sidebarOpen, setSidebarOpen] = useState(true);

  return (
    <div className="flex h-full flex-col">
      {/* ── Top header bar (Vault-style black bar) ─────────── */}
      <header className="flex h-12 shrink-0 items-center justify-between border-b border-neutral-700 bg-neutral-700 px-4">
        <div className="flex items-center gap-3">
          <div className="flex h-7 w-7 items-center justify-center rounded bg-brand text-white">
            <Activity size={14} />
          </div>
          <span className="text-sm font-semibold tracking-wide text-white">
            InfraGraph
          </span>
        </div>
        <div className="flex items-center gap-4">
          <span className="text-xs text-neutral-400">Cluster: local</span>
        </div>
      </header>

      <div className="flex flex-1 overflow-hidden">
        {/* ── Sidebar (Vault-style dark side nav) ────────── */}
        <aside
          className={`${
            sidebarOpen ? 'w-60' : 'w-0'
          } shrink-0 overflow-y-auto overflow-x-hidden border-r border-neutral-200 bg-neutral-900 transition-all duration-200`}
        >
          <div className="px-3 pb-2 pt-4">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-neutral-400">
              Navigation
            </span>
          </div>
          <nav className="flex flex-col gap-0.5 px-2 pb-4">
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.to === '/'}
                className={({ isActive }) =>
                  `group flex items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                    isActive
                      ? 'border-l-2 border-brand bg-white/10 text-white'
                      : 'border-l-2 border-transparent text-neutral-400 hover:bg-white/[0.06] hover:text-neutral-200'
                  }`
                }
              >
                <item.icon size={16} className="shrink-0" />
                {item.label}
              </NavLink>
            ))}
          </nav>
        </aside>

        {/* ── Main content area ──────────────────────────── */}
        <main className="flex flex-1 flex-col overflow-hidden bg-neutral-50">
          {/* Breadcrumbs */}
          <div className="flex items-center gap-1 border-b border-neutral-200 bg-white px-6 py-2 text-sm">
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="mr-2 rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600"
              title={sidebarOpen ? 'Collapse sidebar' : 'Expand sidebar'}
            >
              <ChevronRight
                size={14}
                className={`transition-transform ${sidebarOpen ? 'rotate-180' : ''}`}
              />
            </button>
            {breadcrumbs.map((crumb, i) => (
              <span key={crumb.to} className="flex items-center gap-1">
                {i > 0 && <ChevronRight size={12} className="text-neutral-300" />}
                {i === breadcrumbs.length - 1 ? (
                  <span className="capitalize text-neutral-600">{crumb.label}</span>
                ) : (
                  <NavLink
                    to={crumb.to}
                    className="capitalize text-neutral-400 hover:text-brand"
                  >
                    {crumb.label}
                  </NavLink>
                )}
              </span>
            ))}
          </div>

          {/* Page content */}
          <div className="flex-1 overflow-hidden">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
