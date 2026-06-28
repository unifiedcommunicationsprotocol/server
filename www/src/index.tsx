import { createRoot } from 'react-dom/client';
import { Dashboard } from './components/dashboard/Dashboard';
// @ts-ignore - CSS import
import './styles/globals.css';

const root = document.getElementById('app');
if (root) {
  createRoot(root).render(<Dashboard />);
}
