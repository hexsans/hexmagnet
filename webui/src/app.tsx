import "@/lib/i18n";
import { Routes, Route, Navigate, useLocation, } from "react-router";
import { TooltipProvider, } from "@/components/ui/tooltip";
import { Toaster, } from "@/components/ui/sonner";
import { SiteHeader, } from "@/components/site-header";
import { ThemeProvider, } from "@/components/theme-provider";
import { GraphQLProvider, } from "@/lib/graphql/provider";
import { ErrorBoundary, } from "@/components/error-boundary";
import { BrowsePage, } from "@/pages/browse";
import { DashboardPage, } from "@/pages/dashboard";
import { NotFoundPage, } from "@/pages/not-found";

function AnimatedRoutes() {
  const location = useLocation();
  const key = (location.pathname.split("/",)[1] || "/") + (location.search ? "" : location.key);
  return (
    <div key={key} className="animate-in fade-in slide-in-from-bottom-1 duration-300">
      <Routes>
        <Route path="/" element={<Navigate to="/browse" replace />} />
        <Route path="/browse" element={<BrowsePage />} />
        <Route path="/dashboard" element={<Navigate to="/dashboard/overview" replace />} />
        <Route path="/dashboard/overview" element={<DashboardPage tab="overview" />} />
        <Route path="/dashboard/stats" element={<DashboardPage tab="stats" />} />
        <Route path="/dashboard/config" element={<Navigate to="/dashboard/config/general" replace />} />
        <Route path="/dashboard/config/:configTab" element={<DashboardPage tab="config" />} />
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </div>
  );
}

export function App() {
  return (
    <ThemeProvider defaultTheme="light">
      <GraphQLProvider>
        <TooltipProvider>
          <SiteHeader />
          <ErrorBoundary>
            <AnimatedRoutes />
          </ErrorBoundary>
          <Toaster
            toastOptions={{
              classNames: {
                toast: "font-mono text-xs bg-card border-border text-foreground",
              },
            }}
          />
        </TooltipProvider>
      </GraphQLProvider>
    </ThemeProvider>
  );
}
