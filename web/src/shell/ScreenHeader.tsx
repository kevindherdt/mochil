// shell/ScreenHeader — lets a screen put its toolbar in the shell's non-scrolling header band.
//
// Why this exists: `main` is the scroll container, so anything a screen renders inside `<Page>` scrolls away with
// the content — which is how the library pills vanished when Kevin scrolled his grid ("scrolling down loses the
// upper pills instead of it staying there"). The fix is NOT `position: sticky` in the screen: law L1 says only
// shell/ may position chrome, precisely so each screen can't invent its own scroll behaviour. So the shell owns an
// empty header band above the scroll area, and a screen portals its toolbar into it — the screen still declares
// what the bar contains, the shell decides where it lives and how it behaves.
import { useContext, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { HeaderSlotContext } from "./headerSlot";

export function ScreenHeader({ children }: { children: ReactNode }) {
  const node = useContext(HeaderSlotContext);
  return node ? createPortal(children, node) : null;
}
