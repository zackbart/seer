import { useStdin } from "ink";
import { useEffect, useRef } from "react";

export type ScrollHandler = (direction: "up" | "down", x: number, y: number) => void;

export interface MouseEvent {
  action: "press" | "drag" | "release";
  button: number;
  x: number;
  y: number;
}

export type MouseHandler = (event: MouseEvent) => void;

export function useMouseWheel(handler: ScrollHandler): void {
  useMouse((event) => {
    // Scroll up: btn=64, scroll down: btn=65
    if (event.action === "press" && event.button === 64) {
      handler("up", event.x, event.y);
    } else if (event.action === "press" && event.button === 65) {
      handler("down", event.x, event.y);
    }
  });
}

export function useMouse(handler: MouseHandler): void {
  const { stdin } = useStdin();
  const handlerRef = useRef(handler);
  handlerRef.current = handler;

  useEffect(() => {
    if (!stdin) return;

    // Throttle drag events to avoid per-frame re-renders (~16ms = 60fps cap)
    let lastDragTime = 0;
    const DRAG_THROTTLE_MS = 32; // ~30fps is plenty for text selection

    const onData = (data: Buffer) => {
      const str = data.toString();

      // Parse SGR mouse sequences: \x1b[<btn;x;yM (press) or \x1b[<btn;x;ym (release)
      const sgrMatch = str.match(/\x1b\[<(\d+);(\d+);(\d+)([Mm])/);
      if (sgrMatch) {
        const btn = parseInt(sgrMatch[1]);
        const x = parseInt(sgrMatch[2]);
        const y = parseInt(sgrMatch[3]);
        const isRelease = sgrMatch[4] === "m";

        if (isRelease) {
          handlerRef.current({ action: "release", button: btn & ~0x20, x, y });
        } else if (btn & 0x20) {
          // Motion with button held (bit 5 set = drag) — throttled
          const now = Date.now();
          if (now - lastDragTime < DRAG_THROTTLE_MS) return;
          lastDragTime = now;
          handlerRef.current({ action: "drag", button: btn & ~0x20, x, y });
        } else {
          handlerRef.current({ action: "press", button: btn, x, y });
        }
      }
    };

    stdin.on("data", onData);
    return () => {
      stdin.off("data", onData);
    };
  }, [stdin]);
}
