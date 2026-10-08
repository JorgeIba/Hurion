import { StandingsView, type StandingsDataProps } from '@/components/StandingsView';
import { useScoreboardSearchQuery } from '@/app/routing/useScoreboardSearchQuery';

/** Connects URL search state to the standings without coupling the view to routing. */
export function StandingsRoute(props: StandingsDataProps) {
  const search = useScoreboardSearchQuery();
  return <StandingsView {...props} search={search} />;
}
