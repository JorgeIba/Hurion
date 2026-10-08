import { useEffect } from 'react';
import { useLocation, useNavigationType } from 'react-router';
import { StandingsView, type StandingsDataProps } from '@/components/StandingsView';
import { useScoreboardSearchQuery } from '@/app/routing/useScoreboardSearchQuery';
import { useTabParam } from '@/app/routing/useQueryParam';

/** Connects URL search state to the standings without coupling the view to routing. */
export function StandingsRoute(props: StandingsDataProps) {
  const search = useScoreboardSearchQuery();
  const location = useLocation();
  const navigationType = useNavigationType();
  const [activeTab] = useTabParam();

  useEffect(() => {
    if (
      navigationType === 'PUSH' &&
      location.state?.focusStandings &&
      activeTab === 'standings'
    ) {
      document.getElementById('tab-standings')?.focus({ preventScroll: true });
    }
  }, [location.key, location.state, navigationType, activeTab]);

  return <StandingsView {...props} search={search} />;
}
