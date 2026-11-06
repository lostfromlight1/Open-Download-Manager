import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { App } from "./App";

describe("desktop shell", () => {
  it("shows the product name and that downloads are unavailable", () => {
    render(<App />);

    expect(
      screen.getByRole("heading", { name: "Open Download Manager" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Download management is not available in this build."),
    ).toBeInTheDocument();
  });

  it("does not render fabricated download progress", () => {
    render(<App />);

    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    expect(screen.queryByText(/%/)).not.toBeInTheDocument();
    expect(screen.queryByText(/MB\/s/)).not.toBeInTheDocument();
  });
});
