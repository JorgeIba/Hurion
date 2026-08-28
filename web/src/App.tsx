import { useMemo } from 'react';
import { useScoreboardData } from '@/hooks/useScoreboardData';
import { useClassCodeParam, useTabParam } from '@/hooks/useQueryParam';
import { HeaderBar } from '@/components/HeaderBar';
import { ScoreboardTable } from '@/components/ScoreboardTable';
import { ProblemsView } from '@/components/ProblemsView';
import { AppErrorBoundary } from '@/components/AppErrorBoundary';
import { sortUsersBySolved } from '@/domain/scoreboard/utils';
import { SCOREBOARD_LABELS, SCOREBOARD_CONFIG } from '@/domain/scoreboard/constants';

/**
 * ScoreboardContent Component
 * 
 * Fetches and renders class standings or problem catalog for the active class code:
 * 1. Reads `classCode` from URL search parameters via `useClassCodeParam`.
 * 2. Reads `activeTab` from URL search parameters via `useTabParam` ('standings' | 'problems').
 * 3. Fetches class scoreboard JSON payload via `useScoreboardData`.
 * 4. Sorts students descending by total solved AC count using `sortUsersBySolved`.
 * 5. Renders HeaderBar (with Tab Navigation) and either ScoreboardTable or ProblemsView.
 */
function ScoreboardContent() {
  const [classCode] = useClassCodeParam(SCOREBOARD_CONFIG.DEFAULT_CLASS_CODE);
  const [activeTab, setActiveTab] = useTabParam(SCOREBOARD_CONFIG.DEFAULT_TAB);
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
            <span className="brand-logo">{SCOREBOARD_LABELS.BRAND_LOGO}</span>
            <h1 className="header-title">{SCOREBOARD_CONFIG.DEFAULT_PAGE_TITLE}</h1>
          </div>
        </header>
        <main className="card-container">
          <div className="state-row">{SCOREBOARD_LABELS.LOADING}</div>
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
            <span className="brand-logo">{SCOREBOARD_LABELS.BRAND_LOGO}</span>
            <h1 className="header-title">{SCOREBOARD_CONFIG.DEFAULT_PAGE_TITLE}</h1>
          </div>
        </header>
        <main className="card-container">
          <div className="state-row">{error || SCOREBOARD_LABELS.ERROR_FALLBACK}</div>
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
        activeTab={activeTab}
        onTabChange={setActiveTab}
      />
      <main className="tabpanels-main-container">
        <div className="tabpanel-wrapper" hidden={activeTab !== 'standings'}>
          <ScoreboardTable
            problems={data.problems || []}
            users={sortedUsers}
          />
        </div>
        <div className="tabpanel-wrapper" hidden={activeTab !== 'problems'}>
          <ProblemsView
            problems={data.problems || []}
            users={sortedUsers}
          />
        </div>
      </main>
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

