// app/App.tsx — the one place QueryClient, router and theme providers are wired (plan §5.2).
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "@tanstack/react-router";
import { ShellVariantProvider } from "../shell/ShellVariant";
import { ThemeProvider } from "../features/theme/ThemeContext";
import { PlayerProvider } from "../features/player/PlayerContext";
import { router } from "./router";
import "../styles/tokens.css";
import "../styles/global.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30_000,
    },
  },
});

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <PlayerProvider>
          <ShellVariantProvider>
            <RouterProvider router={router} />
          </ShellVariantProvider>
        </PlayerProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
