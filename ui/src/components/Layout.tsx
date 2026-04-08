import { NavLink, Outlet } from 'react-router-dom';
import { LayoutDashboard, Network, Server, Activity } from 'lucide-react';

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/graph', icon: Network, label: 'Graph' },
  { to: '/resources', icon: Server, label: 'Resources' },
];

export default function Layout() {
  return (
    <div className="app-layout">
      <aside className="sidebar">
        <div className="sidebar-brand">
          <div className="brand-icon">
            <Activity size={16} />
          </div>
          <h1>InfraGraph</h1>
        </div>
        <nav className="sidebar-nav">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) =>
                `sidebar-link${isActive ? ' active' : ''}`
              }
            >
              <item.icon />
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">InfraGraph v0.1.0</div>
      </aside>
      <main className="main-content">
        <Outlet />
      </main>
    </div>
  );
}
