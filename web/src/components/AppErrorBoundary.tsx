import { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
}

/**
 * AppErrorBoundary Component
 *
 * Class-based Error Boundary that catches uncaught rendering exceptions
 * in any child component, preventing full JS white-screen crashes and
 * displaying a graceful Minimal Dark fallback UI with a manual reload option.
 */
export class AppErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
  };

  /**
   * Updates state when an error is thrown in a descendant component.
   */
  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  /**
   * Logs caught error trace details to developer console for diagnostics.
   */
  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Uncaught error in Hurion Scoreboard:', error, errorInfo);
  }

  public render() {
    if (this.state.hasError) {
      return (
        <div className="app-container">
          <header className="header-bar">
            <div className="brand-section">
              <span className="brand-logo">📊</span>
              <h1 className="header-title">Hurion Scoreboard</h1>
            </div>
          </header>
          <main className="card-container">
            <div className="state-row" style={{ color: '#ef4444' }}>
              <h2>⚠️ Something went wrong</h2>
              <p style={{ marginTop: '0.5rem', opacity: 0.8 }}>
                {this.state.error?.message ||
                  'An unexpected error occurred while rendering the scoreboard.'}
              </p>
              <button
                onClick={() => window.location.reload()}
                style={{
                  marginTop: '1rem',
                  padding: '0.5rem 1rem',
                  background: '#121b2d',
                  border: '1px solid rgba(255, 255, 255, 0.15)',
                  color: '#f8fafc',
                  borderRadius: '6px',
                  cursor: 'pointer',
                }}
              >
                Reload Page
              </button>
            </div>
          </main>
        </div>
      );
    }

    return this.props.children;
  }
}
