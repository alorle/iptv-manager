import { describe, expect, it } from "vitest";

import { Badge } from "./badge";

import { render, screen } from "@/test/utils";

describe("Badge", () => {
  it("renders with default variant", () => {
    render(<Badge>Default Badge</Badge>);
    const badge = screen.getByText("Default Badge");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveAttribute("data-slot", "badge");
  });

  it("renders with secondary variant", () => {
    render(<Badge variant="secondary">Secondary Badge</Badge>);
    const badge = screen.getByText("Secondary Badge");
    expect(badge).toBeInTheDocument();
  });

  it("renders with destructive variant", () => {
    render(<Badge variant="destructive">Destructive Badge</Badge>);
    const badge = screen.getByText("Destructive Badge");
    expect(badge).toBeInTheDocument();
  });

  it("renders with outline variant", () => {
    render(<Badge variant="outline">Outline Badge</Badge>);
    const badge = screen.getByText("Outline Badge");
    expect(badge).toBeInTheDocument();
  });

  it("renders with success variant", () => {
    render(<Badge variant="success">Success Badge</Badge>);
    const badge = screen.getByText("Success Badge");
    expect(badge).toBeInTheDocument();
  });

  it("applies custom className", () => {
    render(<Badge className="custom-class">Custom Badge</Badge>);
    const badge = screen.getByText("Custom Badge");
    expect(badge).toHaveClass("custom-class");
  });

  it("renders as span by default", () => {
    render(<Badge>Span Badge</Badge>);
    const badge = screen.getByText("Span Badge");
    expect(badge.tagName).toBe("SPAN");
  });

  it("renders with children elements", () => {
    render(
      <Badge>
        <span>Icon</span> Text
      </Badge>
    );
    expect(screen.getByText("Icon")).toBeInTheDocument();
    expect(screen.getByText(/Text/)).toBeInTheDocument();
  });

  it("passes through additional props", () => {
    render(
      <Badge data-testid="test-badge" title="Test Badge">
        Badge
      </Badge>
    );
    const badge = screen.getByTestId("test-badge");
    expect(badge).toHaveAttribute("title", "Test Badge");
  });
});
