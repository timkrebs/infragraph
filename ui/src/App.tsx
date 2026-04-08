import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import GraphView from './pages/GraphView';
import Resources from './pages/Resources';
import NodeDetail from './pages/NodeDetail';

export default function App() {
  return (
    <BrowserRouter basename="/ui">
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/graph" element={<GraphView />} />
          <Route path="/resources" element={<Resources />} />
          <Route path="/resources/:id" element={<NodeDetail />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
