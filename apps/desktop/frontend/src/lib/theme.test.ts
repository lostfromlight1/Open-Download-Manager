import { describe, expect, it } from "vitest";
import { normalizeTheme } from "./theme";

describe("normalizeTheme", () => {
  it("accepts known modes and defaults unknown to system", () => {
    expect(normalizeTheme("dark")).toBe("dark");
    expect(normalizeTheme("light")).toBe("light");
    expect(normalizeTheme("system")).toBe("system");
    expect(normalizeTheme("nope")).toBe("system");
  });
});
