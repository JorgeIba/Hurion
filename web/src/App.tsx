import { useMemo } from 'react';
import { useScoreboardData } from '@/hooks/useScoreboardData';
import { useClassCodeParam } from '@/hooks/useClassCodeParam';
import { HeaderBar } from '@/components/HeaderBar';
import { ScoreboardTable } from '@/components/ScoreboardTable';
import { AppErrorBoundary } from '@/components/AppErrorBoundary';
import { sortUsersBySolved } from '@/domain/scoreboard/utils';

/**
 * ScoreboardContent Component
 * 
 * Fetches and renders class standings for the active class code:
 * 1. Reads `classCode` from URL search parameters via `useClassCodeParam`.
 * 2. Fetches class scoreboard JSON payload via `useScoreboardData`.
 * 3. Sorts students descending by total solved AC count using `sortUsersBySolved`.
 * 4. Renders HeaderBar and ScoreboardTable components.
 */
function ScoreboardContent() {
  const [classCode] = useClassCodeParam('basicos');
  const { data, loading, error } = useScoreboardData(classCode);

  // Sort students descending by total solved AC count
  const sortedUsers = useMemo(() => {
    if (!data?.users) return [];
    return sortUsersBySolved(data.users);
  }, [data?.users]);

  // Loading UI Skeleton
  if (loading) {
    return (
      <div className="app-container">
        <header className="header-bar">
          <div className="brand-section">
            <span className="brand-logo">📊</span>
            <h1 className="header-title">Scoreboard</h1>
          </div>
        </header>
        <main className="card-container">
          <div className="state-row">Loading scoreboard data...</div>
        </main>
      </div>
    );
  }

  // Error UI Guard (Network or HTTP 404)
  if (error || !data) {
    return (
      <div className="app-container">
        <header className="header-bar">
          <div className="brand-section">
            <span className="brand-logo">📊</span>
            <h1 className="header-title">Scoreboard</h1>
          </div>
        </header>
        <main className="card-container">
          <div className="state-row">{error || 'Failed to load scoreboard'}</div>
        </main>
      </div>
    );
  }

  return (
    <div className="app-container">
      <HeaderBar
        title={data.className || data.classCode}
        users={sortedUsers}
        problems={data.problems || []}
      />
      <ScoreboardTable
        problems={data.problems || []}
        users={sortedUsers}
      />
    </div>
  );
}

/**
 * Main App Component
 * Wraps ScoreboardContent inside AppErrorBoundary to catch uncaught runtime exceptions.
 */
export function App() {
  return (
    <AppErrorBoundary>
      <ScoreboardContent />
    </AppErrorBoundary>
  );
}

export default App;
