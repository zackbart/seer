import { useInput } from "ink";
import { useRef } from "react";

export type KeyHandler = (key: string, raw: string) => void;

export function useKeyBindings(handler: KeyHandler): void {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;

  useInput((input, key) => {
    let keyStr = "";

    if (key.ctrl && input === "d") keyStr = "ctrl+d";
    else if (key.ctrl && input === "u") keyStr = "ctrl+u";
    else if (key.ctrl && input === "c") keyStr = "ctrl+c";
    else if (key.escape) keyStr = "esc";
    else if (key.return) keyStr = "enter";
    else if (key.backspace || key.delete) keyStr = "backspace";
    else if (key.upArrow) keyStr = key.meta ? "alt+up" : "up";
    else if (key.downArrow) keyStr = key.meta ? "alt+down" : "down";
    else if (key.leftArrow) keyStr = key.meta ? "alt+left" : "left";
    else if (key.rightArrow) keyStr = key.meta ? "alt+right" : "right";
    else if (key.pageDown) keyStr = "pagedown";
    else if (key.pageUp) keyStr = "pageup";
    else if (key.tab) keyStr = "tab";
    else if (input === " ") keyStr = "space";
    else if (input.length === 1) keyStr = input;

    if (keyStr) {
      handlerRef.current(keyStr, input);
    }
  });
}
