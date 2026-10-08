import { createBrowserRouter, createRoutesFromElements, Route } from 'react-router';
import { RouterProvider } from 'react-router/dom';
import { ScoreboardPage } from '@/app/ScoreboardPage';

const appRouter = createBrowserRouter(
  createRoutesFromElements(
    // View selection remains in URL parameters; all paths retain the existing page.
    <Route path="*" element={<ScoreboardPage />} />
  )
);

export function AppRouter() {
  return <RouterProvider router={appRouter} />;
}
