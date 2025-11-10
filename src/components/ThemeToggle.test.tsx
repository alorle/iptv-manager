import { beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeToggle } from "./ThemeToggle";

import { render, screen, userEvent } from "@/test/utils";

describe("ThemeToggle", () => {
  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear();

    // Mock matchMedia
    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: vi.fn().mockImplementation((query) => ({
        matches: false,
        // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    });

    // Reset document classes
    document.documentElement.classList.remove("dark");
  });

  it("renders toggle button with aria-label", () => {
    render(<ThemeToggle />);
    const button = screen.getByRole("button", { name: /toggle theme/i });
    expect(button).toBeInTheDocument();
  });

  it("shows Moon icon in light mode", () => {
    render(<ThemeToggle />);
    const button = screen.getByRole("button", { name: /toggle theme/i });
    // Moon icon should be present when theme is light
    expect(button.querySelector("svg")).toBeInTheDocument();
  });

  it("toggles theme when clicked", async () => {
    const user = userEvent.setup();
    render(<ThemeToggle />);

    const button = screen.getByRole("button", { name: /toggle theme/i });

    // Initially light mode (no dark class)
    expect(document.documentElement.classList.contains("dark")).toBe(false);

    // Click to toggle to dark mode
    await user.click(button);

    // Should add dark class
    expect(document.documentElement.classList.contains("dark")).toBe(true);

    // Click again to toggle back to light mode
    await user.click(button);

    // Should remove dark class
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("persists theme preference to localStorage", async () => {
    const user = userEvent.setup();
    render(<ThemeToggle />);

    const button = screen.getByRole("button", { name: /toggle theme/i });

    // Initial theme should be stored
    expect(localStorage.getItem("theme")).toBe("light");

    // Toggle to dark
    await user.click(button);
    expect(localStorage.getItem("theme")).toBe("dark");

    // Toggle back to light
    await user.click(button);
    expect(localStorage.getItem("theme")).toBe("light");
  });

  it("reads initial theme from localStorage", () => {
    localStorage.setItem("theme", "dark");
    render(<ThemeToggle />);

    // Should start with dark mode
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("uses system preference when localStorage is empty", () => {
    // Mock dark mode preference
    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: vi.fn().mockImplementation((query) => ({
        matches: query === "(prefers-color-scheme: dark)",
        // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    });

    render(<ThemeToggle />);

    // Should start with dark mode based on system preference
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });
});
