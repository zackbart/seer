import { describe, expect, test } from "bun:test";
import {
  computeWrappedBody,
  computeWrappedBodyFromLines,
  sanitizeTerminalText,
  stripAnsi,
  wrapAnsiText,
} from "./ansiText.js";

describe("sanitizeTerminalText", () => {
  test("preserves SGR styling but removes cursor-control escapes", () => {
    const input = "\x1b[31mred\x1b[0m\x1b[2K\x1b[10Cafter";
    expect(sanitizeTerminalText(input)).toBe("\x1b[31mred\x1b[0mafter");
  });

  test("normalizes control characters that commonly appear in extracted docs", () => {
    const input = "a\rb\t\f\x08c\n";
    expect(sanitizeTerminalText(input)).toBe("a\nb    \nc\n");
    expect(sanitizeTerminalText(input, { preserveTabs: true })).toBe("a\nb\t\nc\n");
  });
});

describe("stripAnsi", () => {
  test("removes non-SGR CSI escapes as well as styling", () => {
    const input = "\x1b[31mred\x1b[0m\x1b[2K\x1b[10Cdone";
    expect(stripAnsi(input)).toBe("reddone");
  });
});

describe("computeWrappedBodyFromLines", () => {
  test("matches computeWrappedBody when wrapping is supplied by caller", () => {
    const text = "alpha beta gamma delta\n\x1b[31mred green blue\x1b[0m";
    const wrapped = wrapAnsiText(text, 10);

    expect(computeWrappedBodyFromLines(wrapped, 3, 1)).toEqual(
      computeWrappedBody(text, 10, 3, 1),
    );
  });
});
