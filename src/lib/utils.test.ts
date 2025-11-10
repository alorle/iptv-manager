import { describe, expect, it } from "vitest";

import { cn, toTitleCase } from "./utils";

describe("cn", () => {
  it("merges class names correctly", () => {
    expect(cn("foo", "bar")).toBe("foo bar");
  });

  it("handles conditional classes", () => {
    // eslint-disable-next-line no-constant-binary-expression
    expect(cn("foo", false && "bar", "baz")).toBe("foo baz");
  });

  it("merges Tailwind classes correctly", () => {
    expect(cn("px-2 py-1", "px-4")).toBe("py-1 px-4");
  });

  it("handles empty inputs", () => {
    expect(cn()).toBe("");
  });

  it("handles undefined and null", () => {
    expect(cn("foo", undefined, null, "bar")).toBe("foo bar");
  });

  it("handles arrays", () => {
    expect(cn(["foo", "bar"], "baz")).toBe("foo bar baz");
  });

  it("handles objects with boolean values", () => {
    expect(cn({ foo: true, bar: false, baz: true })).toBe("foo baz");
  });
});

describe("toTitleCase", () => {
  it("converts lowercase to title case", () => {
    expect(toTitleCase("hello world")).toBe("Hello World");
  });

  it("converts uppercase to title case", () => {
    expect(toTitleCase("HELLO WORLD")).toBe("Hello World");
  });

  it("converts mixed case to title case", () => {
    expect(toTitleCase("hElLo WoRlD")).toBe("Hello World");
  });

  it("handles single word", () => {
    expect(toTitleCase("hello")).toBe("Hello");
  });

  it("handles empty string", () => {
    expect(toTitleCase("")).toBe("");
  });

  it("handles strings with multiple spaces", () => {
    expect(toTitleCase("hello  world")).toBe("Hello  World");
  });

  it("handles hyphenated words", () => {
    // The function treats hyphens as part of the word, not as boundaries
    expect(toTitleCase("hello-world")).toBe("Hello-world");
  });
});
