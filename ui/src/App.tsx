import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import GraphView from './pages/GraphView';
import Resources from './pages/Resources';
import NodeDetail from './pages/NodeDetail';
import ImpactAnalysis from './pages/ImpactAnalysis';
import Collectors from './pages/Collectors';
import Settings from './pages/Settings';

export default function App() {
  return (
    <BrowserRouter basename="/ui">
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/resources" element={<Resources />} />
          <Route path="/resources/:id" element={<NodeDetail />} />
          <Route path="/graph" element={<GraphView />} />
          <Route path="/impact" element={<ImpactAnalysis />} />
          <Route path="/collectors" element={<Collectors />} />
          <Route path="/settings" element={<Settings />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
